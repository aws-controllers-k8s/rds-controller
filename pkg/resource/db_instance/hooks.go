// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package db_instance

import (
	"context"
	"errors"
	"fmt"
	"strings"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/rds"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	corev1 "k8s.io/api/core/v1"

	"github.com/aws-controllers-k8s/rds-controller/pkg/util"
)

// NOTE(jaypipes): The below list is derived from looking at the RDS control
// plane source code. I've spoken with the RDS team about not having a list of
// publicly-visible DBInstanceStatus strings available through the API model
// and they are looking into doing that. If we get that, we could use the
// constants generated in the `apis/{VERSION}/enums.go` file.
const (
	StatusAvailable                                    = "available"
	StatusBackingUp                                    = "backing-up"
	StatusCreating                                     = "creating"
	StatusDeleted                                      = "deleted"
	StatusDeleting                                     = "deleting"
	StatusFailed                                       = "failed"
	StatusBacktracking                                 = "backtracking"
	StatusModifying                                    = "modifying"
	StatusUpgrading                                    = "upgrading"
	StatusRebooting                                    = "rebooting"
	StatusResettingMasterCredentials                   = "resetting-master-credentials"
	StatusStorageFull                                  = "storage-full"
	StatusIncompatibleCredentials                      = "incompatible-credentials"
	StatusIncompatibleOptionGroup                      = "incompatible-option-group"
	StatusIncompatibleParameters                       = "incompatible-parameters"
	StatusIncompatibleRestore                          = "incompatible-restore"
	StatusIncompatibleNetwork                          = "incompatible-network"
	StatusRestoreError                                 = "restore-error"
	StatusRenaming                                     = "renaming"
	StatusInaccessibleEncryptionCredentialsRecoverable = "inaccessible-encryption-credentials-recoverable"
	StatusInaccessibleEncryptionCredentials            = "inaccessible-encryption-credentials"
	StatusMaintenance                                  = "maintenance"
	StatusConfiguringEnhancedMonitoring                = "configuring-enhanced-monitoring"
	StatusStopping                                     = "stopping"
	StatusStopped                                      = "stopped"
	StatusStarting                                     = "starting"
	StatusMovingToVPC                                  = "moving-to-vpc"
	StatusConvertingToVPC                              = "converting-to-vpc"
	StatusConfiguringIAMDatabaseAuthr                  = "configuring-iam-database-auth"
	StatusStorageOptimization                          = "storage-optimization"
	StatusConfiguringLogExports                        = "configuring-log-exports"
	StatusConfiguringAssociatedRoles                   = "configuring-associated-roles"
	StatusConfiguringActivityStream                    = "configuring-activity-stream"
	StatusInsufficientCapacity                         = "insufficient-capacity"
	StatusValidatingConfiguration                      = "validating-configuration"
	StatusUnsupportedConfiguration                     = "unsupported-configuration"
	StatusAutomationPaused                             = "automation-paused"
)

var (
	ServiceDefaultBackupTarget            = "region"
	ServiceDefaultNetworkType             = "IPV4"
	ServiceDefaultInsightsRetentionPeriod = int64(7)
)

var (
	// TerminalStatuses are the status strings that are terminal states for a
	// DB instance.
	TerminalStatuses = []string{
		StatusDeleted,
		StatusDeleting,
		StatusInaccessibleEncryptionCredentialsRecoverable,
		StatusInaccessibleEncryptionCredentials,
		StatusIncompatibleNetwork,
		StatusIncompatibleRestore,
		StatusFailed,
	}
	// UnableToFinalSnapshotStatuses are those status strings that indicate a
	// DB instance cannot have a final snapshot taken before deletion.
	UnableToFinalSnapshotStatuses = []string{
		StatusIncompatibleRestore,
		StatusFailed,
	}
)

var (
	requeueWaitWhileDeleting = ackrequeue.NeededAfter(
		errors.New("DB instance in 'deleting' state, cannot be modified or deleted."),
		ackrequeue.DefaultRequeueAfterDuration,
	)
)

// requeueWaitUntilCanModify returns a `ackrequeue.RequeueNeededAfter` struct
// explaining the DB instance cannot be modified until it reaches an available
// status.
func requeueWaitUntilCanModify(r *resource) *ackrequeue.RequeueNeededAfter {
	if r.ko.Status.DBInstanceStatus == nil {
		return nil
	}
	status := *r.ko.Status.DBInstanceStatus
	msg := fmt.Sprintf(
		"DB Instance in '%s' state, cannot be modified until '%s'.",
		status, StatusAvailable,
	)
	return ackrequeue.NeededAfter(
		errors.New(msg),
		ackrequeue.DefaultRequeueAfterDuration,
	)
}

// instanceHasTerminalStatus returns whether the supplied DB Instance is in a
// terminal state
func instanceHasTerminalStatus(r *resource) bool {
	if r.ko.Status.DBInstanceStatus == nil {
		return false
	}
	dbis := *r.ko.Status.DBInstanceStatus
	for _, s := range TerminalStatuses {
		if dbis == s {
			return true
		}
	}
	return false
}

// instanceAvailable returns true if the supplied DB instance is in an
// available status
func instanceAvailable(r *resource) bool {
	if r.ko.Status.DBInstanceStatus == nil {
		return false
	}
	dbis := *r.ko.Status.DBInstanceStatus
	return dbis == StatusAvailable
}

// instanceCreating returns true if the supplied DB instance is in the process
// of being created
func instanceCreating(r *resource) bool {
	if r.ko.Status.DBInstanceStatus == nil {
		return false
	}
	dbis := *r.ko.Status.DBInstanceStatus
	return dbis == StatusCreating
}

// instanceDeleting returns true if the supplied DB instance is in the process
// of being deleted
func instanceDeleting(r *resource) bool {
	if r.ko.Status.DBInstanceStatus == nil {
		return false
	}
	dbis := *r.ko.Status.DBInstanceStatus
	return dbis == StatusDeleting
}

func customPreCompare(delta *ackcompare.Delta, a *resource, b *resource) {
	if a.ko.Spec.PerformanceInsightsKMSKeyID == nil {
		a.ko.Spec.PerformanceInsightsKMSKeyID = b.ko.Spec.PerformanceInsightsKMSKeyID
	}

	if a.ko.Spec.PerformanceInsightsRetentionPeriod == nil {
		a.ko.Spec.PerformanceInsightsRetentionPeriod = b.ko.Spec.PerformanceInsightsRetentionPeriod
	}
}

// function to create restoreDbInstanceFromDbSnapshot payload and call restoreDbInstanceFromDbSnapshot API
func (rm *resourceManager) restoreDbInstanceFromDbSnapshot(
	ctx context.Context,
	r *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.restoreDbInstanceFromDbSnapshot")
	defer func(err error) { exit(err) }(err)

	input, err := rm.newRestoreDBInstanceFromDBSnapshotInput(r)
	if err != nil {
		return nil, err
	}
	resp, respErr := rm.sdkapi.RestoreDBInstanceFromDBSnapshot(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "RestoreDbInstanceFromDbSnapshot", respErr)
	if respErr != nil {
		return nil, respErr
	}

	rm.setResourceFromRestoreDBInstanceFromDBSnapshotOutput(r, resp)
	rm.setStatusDefaults(r.ko)

	// We expect the DB instance to be in 'creating' status since we just
	// issued the call to create it, but I suppose it doesn't hurt to check
	// here.
	if instanceCreating(&resource{r.ko}) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(&resource{r.ko}, corev1.ConditionFalse, nil, nil)
	}
	return &resource{r.ko}, nil
}

// newCreateDBInstanceReadReplicaInput returns a CreateDBInstanceReadReplicaInput object
// with each the field set by the corresponding configuration's fields.
// We copy the function here because currently we don't have logic to rename param
// that went through this call, espeically we want to use PerformanceInsightsEnabled instead of
// EnablePerformanceInsights here
// we will remove this part when code generator can have logic to add rename in `GoCodeSetSDKForStruct`
func newCreateDBInstanceReadReplicaInput(
	r *resource,
) *svcsdk.CreateDBInstanceReadReplicaInput {
	res := &svcsdk.CreateDBInstanceReadReplicaInput{}

	if r.ko.Spec.AutoMinorVersionUpgrade != nil {
		res.AutoMinorVersionUpgrade = r.ko.Spec.AutoMinorVersionUpgrade
	}
	if r.ko.Spec.AvailabilityZone != nil {
		res.AvailabilityZone = r.ko.Spec.AvailabilityZone
	}
	if r.ko.Spec.CopyTagsToSnapshot != nil {
		res.CopyTagsToSnapshot = r.ko.Spec.CopyTagsToSnapshot
	}
	if r.ko.Spec.CustomIAMInstanceProfile != nil {
		res.CustomIamInstanceProfile = r.ko.Spec.CustomIAMInstanceProfile
	}
	if r.ko.Spec.DBInstanceClass != nil {
		res.DBInstanceClass = r.ko.Spec.DBInstanceClass
	}
	if r.ko.Spec.DBInstanceIdentifier != nil {
		res.DBInstanceIdentifier = r.ko.Spec.DBInstanceIdentifier
	}
	if r.ko.Spec.DBParameterGroupName != nil {
		res.DBParameterGroupName = r.ko.Spec.DBParameterGroupName
	}
	if r.ko.Spec.DBSubnetGroupName != nil {
		res.DBSubnetGroupName = r.ko.Spec.DBSubnetGroupName
	}
	if r.ko.Spec.DeletionProtection != nil {
		res.DeletionProtection = r.ko.Spec.DeletionProtection
	}
	// TODO: michaelhtm Unsure what to do with this field
	// if r.ko.Spec.DestinationRegion != nil {
	// 	res.destinationRegion = r.ko.Spec.DestinationRegion
	// }
	if r.ko.Spec.Domain != nil {
		res.Domain = r.ko.Spec.Domain
	}
	if r.ko.Spec.DomainIAMRoleName != nil {
		res.DomainIAMRoleName = r.ko.Spec.DomainIAMRoleName
	}
	if r.ko.Spec.EnableCloudwatchLogsExports != nil {
		res.EnableCloudwatchLogsExports = aws.ToStringSlice(r.ko.Spec.EnableCloudwatchLogsExports)
	}
	if r.ko.Spec.EnableIAMDatabaseAuthentication != nil {
		res.EnableIAMDatabaseAuthentication = r.ko.Spec.EnableIAMDatabaseAuthentication
	}
	if r.ko.Spec.PerformanceInsightsEnabled != nil {
		res.EnablePerformanceInsights = r.ko.Spec.PerformanceInsightsEnabled
	}
	if r.ko.Spec.IOPS != nil {
		res.Iops = aws.Int32(int32(*r.ko.Spec.IOPS))
	}
	if r.ko.Spec.KMSKeyID != nil {
		res.KmsKeyId = r.ko.Spec.KMSKeyID
	}
	if r.ko.Spec.MaxAllocatedStorage != nil {
		res.MaxAllocatedStorage = aws.Int32(int32(*r.ko.Spec.MaxAllocatedStorage))
	}
	if r.ko.Spec.MonitoringInterval != nil {
		res.MonitoringInterval = aws.Int32(int32(*r.ko.Spec.MonitoringInterval))
	}
	if r.ko.Spec.MonitoringRoleARN != nil {
		res.MonitoringRoleArn = r.ko.Spec.MonitoringRoleARN
	}
	if r.ko.Spec.MultiAZ != nil {
		res.MultiAZ = r.ko.Spec.MultiAZ
	}
	if r.ko.Spec.NetworkType != nil {
		res.NetworkType = r.ko.Spec.NetworkType
	}
	if r.ko.Spec.OptionGroupName != nil {
		res.OptionGroupName = r.ko.Spec.OptionGroupName
	}
	if r.ko.Spec.PerformanceInsightsKMSKeyID != nil {
		res.PerformanceInsightsKMSKeyId = r.ko.Spec.PerformanceInsightsKMSKeyID
	}
	if r.ko.Spec.PerformanceInsightsRetentionPeriod != nil {
		res.PerformanceInsightsRetentionPeriod = aws.Int32(int32(*r.ko.Spec.PerformanceInsightsRetentionPeriod))
	}
	if r.ko.Spec.Port != nil {
		res.Port = aws.Int32(int32(*r.ko.Spec.Port))
	}
	if r.ko.Spec.PreSignedURL != nil {
		res.PreSignedUrl = r.ko.Spec.PreSignedURL
	}
	if r.ko.Spec.ProcessorFeatures != nil {
		resf27 := []svcsdktypes.ProcessorFeature{}
		for _, resf27iter := range r.ko.Spec.ProcessorFeatures {
			resf27elem := svcsdktypes.ProcessorFeature{}
			if resf27iter.Name != nil {
				resf27elem.Name = resf27iter.Name
			}
			if resf27iter.Value != nil {
				resf27elem.Value = resf27iter.Value
			}
			resf27 = append(resf27, resf27elem)
		}
		res.ProcessorFeatures = resf27
	}
	if r.ko.Spec.PubliclyAccessible != nil {
		res.PubliclyAccessible = r.ko.Spec.PubliclyAccessible
	}
	if r.ko.Spec.ReplicaMode != nil {
		res.ReplicaMode = svcsdktypes.ReplicaMode(*r.ko.Spec.ReplicaMode)
	}
	if r.ko.Spec.SourceDBInstanceIdentifier != nil {
		res.SourceDBInstanceIdentifier = r.ko.Spec.SourceDBInstanceIdentifier
	}
	if r.ko.Spec.SourceRegion != nil {
		res.SourceRegion = r.ko.Spec.SourceRegion
	}
	if r.ko.Spec.StorageType != nil {
		res.StorageType = r.ko.Spec.StorageType
	}
	if r.ko.Spec.Tags != nil {
		resf33 := []svcsdktypes.Tag{}
		for _, resf33iter := range r.ko.Spec.Tags {
			resf33elem := svcsdktypes.Tag{}
			if resf33iter.Key != nil {
				resf33elem.Key = resf33iter.Key
			}
			if resf33iter.Value != nil {
				resf33elem.Value = resf33iter.Value
			}
			resf33 = append(resf33, resf33elem)
		}
		res.Tags = resf33
	}
	if r.ko.Spec.UseDefaultProcessorFeatures != nil {
		res.UseDefaultProcessorFeatures = r.ko.Spec.UseDefaultProcessorFeatures
	}
	if r.ko.Spec.VPCSecurityGroupIDs != nil {
		res.VpcSecurityGroupIds = aws.ToStringSlice(r.ko.Spec.VPCSecurityGroupIDs)
	}

	return res
}

// function to create createDBInstanceReadReplica payload and call createDBInstanceReadReplica API
func (rm *resourceManager) createDBInstanceReadReplica(
	ctx context.Context,
	r *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.createDBInstanceReadReplica")
	defer func(err error) { exit(err) }(err)

	resp, respErr := rm.sdkapi.CreateDBInstanceReadReplica(ctx, newCreateDBInstanceReadReplicaInput(r))
	rm.metrics.RecordAPICall("CREATE", "CreateDBInstanceReadReplica", respErr)
	if respErr != nil {
		return nil, respErr
	}

	rm.setResourceFromCreateDBInstanceReadReplicaOutput(r, resp)
	rm.setStatusDefaults(r.ko)

	// We expect the DB instance to be in 'creating' status since we just
	// issued the call to create it, but I suppose it doesn't hurt to check
	// here.
	if instanceCreating(&resource{r.ko}) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(&resource{r.ko}, corev1.ConditionFalse, nil, nil)
	}
	return &resource{r.ko}, nil
}

// RDS will choose preferred engine minor version if only
// engine major version is provided and controler should not
// treat them as different, such as spec has 14, status has 14.1
// controller should treat them as same
func reconcileEngineVersion(
	a *resource,
	b *resource,
) {
	if a != nil && b != nil && a.ko.Spec.EngineVersion != nil && b.ko.Spec.EngineVersion != nil && strings.HasPrefix(*b.ko.Spec.EngineVersion, *a.ko.Spec.EngineVersion) {
		a.ko.Spec.EngineVersion = b.ko.Spec.EngineVersion
	}
}

// syncTags keeps the resource's tags in sync
//
// NOTE(jaypipes): RDS' Tagging APIs differ from other AWS APIs in the
// following ways:
//
//  1. The names of the tagging API operations are different. Other APIs use the
//     Tagris `ListTagsForResource`, `TagResource` and `UntagResource` API
//     calls. RDS uses `ListTagsForResource`, `AddTagsToResource` and
//     `RemoveTagsFromResource`.
//
//  2. Even though the name of the `ListTagsForResource` API call is the same,
//     the structure of the input and the output are different from other APIs.
//     For the input, instead of a `ResourceArn` field, RDS names the field
//     `ResourceName`, but actually expects an ARN, not the instance
//     name.  This is the same for the `AddTagsToResource` and
//     `RemoveTagsFromResource` input shapes. For the output shape, the field is
//     called `TagList` instead of `Tags` but is otherwise the same struct with
//     a `Key` and `Value` member field.
func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncTags")
	defer func() { exit(err) }()

	arn := (*string)(latest.ko.Status.ACKResourceMetadata.ARN)

	toAdd, toDelete := util.ComputeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from instance", "tags", toDelete)
		_, err = rm.sdkapi.RemoveTagsFromResource(
			ctx,
			&svcsdk.RemoveTagsFromResourceInput{
				ResourceName: arn,
				TagKeys:      toDelete,
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "RemoveTagsFromResource", err)
		if err != nil {
			return err
		}
	}

	// NOTE(jaypipes): According to the RDS API documentation, adding a tag
	// with a new value overwrites any existing tag with the same key. So, we
	// don't need to do anything to "update" a Tag. Simply including it in the
	// AddTagsToResource call is enough.
	if len(toAdd) > 0 {
		rlog.Debug("adding tags to instance", "tags", toAdd)
		_, err = rm.sdkapi.AddTagsToResource(
			ctx,
			&svcsdk.AddTagsToResourceInput{
				ResourceName: arn,
				Tags:         sdkTagsFromResourceTags(toAdd),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "AddTagsToResource", err)
		if err != nil {
			return err
		}
	}
	return nil
}

// getTags retrieves the resource's associated tags
func (rm *resourceManager) getTags(
	ctx context.Context,
	resourceARN string,
) ([]*svcapitypes.Tag, error) {
	resp, err := rm.sdkapi.ListTagsForResource(
		ctx,
		&svcsdk.ListTagsForResourceInput{
			ResourceName: &resourceARN,
		},
	)
	rm.metrics.RecordAPICall("GET", "ListTagsForResource", err)
	if err != nil {
		return nil, err
	}
	tags := make([]*svcapitypes.Tag, 0, len(resp.TagList))
	for _, tag := range resp.TagList {
		tags = append(tags, &svcapitypes.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return tags, nil
}

// compareTags adds a difference to the delta if the supplied resources have
// different tag collections
func compareTags(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if len(a.ko.Spec.Tags) != len(b.ko.Spec.Tags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	} else if len(a.ko.Spec.Tags) > 0 {
		if !util.EqualTags(a.ko.Spec.Tags, b.ko.Spec.Tags) {
			delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
		}
	}
}

// sdkTagsFromResourceTags transforms a *svcapitypes.Tag array to a *svcsdk.Tag
// array.
func sdkTagsFromResourceTags(
	rTags []*svcapitypes.Tag,
) []svcsdktypes.Tag {
	tags := make([]svcsdktypes.Tag, len(rTags))
	for i := range rTags {
		tags[i] = svcsdktypes.Tag{
			Key:   rTags[i].Key,
			Value: rTags[i].Value,
		}
	}
	return tags
}

// TODO(a-hilaly): generate this code.

// getLastAppliedSecretReferenceString returns a string representation of the
// last-applied secret reference.
func getLastAppliedSecretReferenceString(r *v1alpha1.SecretKeyReference) string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s.%s", r.Namespace, r.Name, r.Key)
}

// setLastAppliedSecretReferenceAnnotation sets the last-applied secret reference
// annotation on the supplied resource.
func setLastAppliedSecretReferenceAnnotation(r *resource) {
	if r.ko.Annotations == nil {
		r.ko.Annotations = make(map[string]string)
	}
	r.ko.Annotations[svcapitypes.LastAppliedSecretAnnotation] = getLastAppliedSecretReferenceString(r.ko.Spec.MasterUserPassword)
}

// getLastAppliedSecretReferenceAnnotation returns the last-applied secret reference
// annotation on the supplied resource.
func getLastAppliedSecretReferenceAnnotation(r *resource) string {
	if r.ko.Annotations == nil {
		return ""
	}
	return r.ko.Annotations[svcapitypes.LastAppliedSecretAnnotation]
}

func compareSecretReferenceChanges(
	delta *ackcompare.Delta,
	desired *resource,
	latest *resource,
) {
	oldRef := getLastAppliedSecretReferenceAnnotation(desired)
	newRef := getLastAppliedSecretReferenceString(desired.ko.Spec.MasterUserPassword)
	if oldRef != newRef {
		delta.Add("Spec.MasterUserPassword", oldRef, newRef)
	}
}

// setDeleteDBInstanceInput uses the resource annotations to complete
// the input for the DeleteDBInstance API call.
func setDeleteDBInstanceInput(
	r *resource,
	input *svcsdk.DeleteDBInstanceInput,
) error {
	params, err := util.ParseDeletionAnnotations(r.ko.GetAnnotations())
	if err != nil {
		return err
	}
	input.SkipFinalSnapshot = params.SkipFinalSnapshot
	input.FinalDBSnapshotIdentifier = params.FinalDBSnapshotIdentifier
	input.DeleteAutomatedBackups = params.DeleteAutomatedBackup
	return nil
}

// needStorageUpdate
func needStorageUpdate(
	r *resource,
	delta *ackcompare.Delta,
) bool {
	return strings.Contains(*r.ko.Status.DBInstanceStatus, "storage-full") &&
		delta.DifferentAt("Spec.AllocatedStorage")
}
