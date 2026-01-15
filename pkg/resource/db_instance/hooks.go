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
	"slices"
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

func customPreCompare(delta *ackcompare.Delta, a *resource, b *resource) {
	// Do not consider any of the following fields for delta if they are missing in
	// desired(a) but are present in latest(b) because each of these fields is
	// late-initialized
	// This special handling is only needed for DBInstance because late
	// initialized values are not returned after successful ModifyDBInstance
	// call. They are only populated once the DBInstance returns back to
	// available.
	if a.ko.Spec.AvailabilityZone == nil &&
		b.ko.Spec.AvailabilityZone != nil {
		a.ko.Spec.AvailabilityZone = b.ko.Spec.AvailabilityZone
	}
	if a.ko.Spec.BackupTarget == nil &&
		b.ko.Spec.BackupTarget != nil &&
		*b.ko.Spec.BackupTarget == ServiceDefaultBackupTarget {
		a.ko.Spec.BackupTarget = b.ko.Spec.BackupTarget
	}
	if a.ko.Spec.NetworkType == nil &&
		b.ko.Spec.NetworkType != nil &&
		*b.ko.Spec.NetworkType == ServiceDefaultNetworkType {
		a.ko.Spec.NetworkType = b.ko.Spec.NetworkType
	}
	if a.ko.Spec.PerformanceInsightsEnabled == nil &&
		b.ko.Spec.PerformanceInsightsEnabled != nil {
		a.ko.Spec.PerformanceInsightsEnabled = aws.Bool(false)
	}

	// RDS will choose preferred engine minor version if only
	// engine major version is provided and controler should not
	// treat them as different, such as spec has 14, status has 14.1
	// controller should treat them as same
	reconcileEngineVersion(a, b)
	compareTags(delta, a, b)
	compareSecretReferenceChanges(delta, a, b)

	// if dbinstances are created from a dbcluster, certain fields can only be changed on dbclusters,
	// not in dbinstances.
	// With the following change, we ensure we don't try to update the following fields id DBClusterIdentifier
	// is defined.
	if a.ko.Spec.DBClusterIdentifier == nil {
		if ackcompare.HasNilDifference(a.ko.Spec.DatabaseInsightsMode, b.ko.Spec.DatabaseInsightsMode) {
			delta.Add("Spec.DatabaseInsightsMode", a.ko.Spec.DatabaseInsightsMode, b.ko.Spec.DatabaseInsightsMode)
		} else if a.ko.Spec.DatabaseInsightsMode != nil && b.ko.Spec.DatabaseInsightsMode != nil {
			if *a.ko.Spec.DatabaseInsightsMode != *b.ko.Spec.DatabaseInsightsMode {
				delta.Add("Spec.DatabaseInsightsMode", a.ko.Spec.DatabaseInsightsMode, b.ko.Spec.DatabaseInsightsMode)
			}
		}

		if len(a.ko.Spec.EnableCloudwatchLogsExports) != len(b.ko.Spec.EnableCloudwatchLogsExports) {
			delta.Add("Spec.EnableCloudwatchLogsExports", a.ko.Spec.EnableCloudwatchLogsExports, b.ko.Spec.EnableCloudwatchLogsExports)
		} else if len(a.ko.Spec.EnableCloudwatchLogsExports) > 0 {
			if !ackcompare.SliceStringPEqual(a.ko.Spec.EnableCloudwatchLogsExports, b.ko.Spec.EnableCloudwatchLogsExports) {
				delta.Add("Spec.EnableCloudwatchLogsExports", a.ko.Spec.EnableCloudwatchLogsExports, b.ko.Spec.EnableCloudwatchLogsExports)
			}
		}

		if ackcompare.HasNilDifference(a.ko.Spec.MaxAllocatedStorage, b.ko.Spec.MaxAllocatedStorage) {
			delta.Add("Spec.MaxAllocatedStorage", a.ko.Spec.MaxAllocatedStorage, b.ko.Spec.MaxAllocatedStorage)
		} else if a.ko.Spec.MaxAllocatedStorage != nil && b.ko.Spec.MaxAllocatedStorage != nil {
			if *a.ko.Spec.MaxAllocatedStorage != *b.ko.Spec.MaxAllocatedStorage {
				delta.Add("Spec.MaxAllocatedStorage", a.ko.Spec.MaxAllocatedStorage, b.ko.Spec.MaxAllocatedStorage)
			}
		}
		if ackcompare.HasNilDifference(a.ko.Spec.BackupRetentionPeriod, b.ko.Spec.BackupRetentionPeriod) {
			delta.Add("Spec.BackupRetentionPeriod", a.ko.Spec.BackupRetentionPeriod, b.ko.Spec.BackupRetentionPeriod)
		} else if a.ko.Spec.BackupRetentionPeriod != nil && b.ko.Spec.BackupRetentionPeriod != nil {
			if *a.ko.Spec.BackupRetentionPeriod != *b.ko.Spec.BackupRetentionPeriod {
				delta.Add("Spec.BackupRetentionPeriod", a.ko.Spec.BackupRetentionPeriod, b.ko.Spec.BackupRetentionPeriod)
			}
		}

		if ackcompare.HasNilDifference(a.ko.Spec.PreferredBackupWindow, b.ko.Spec.PreferredBackupWindow) {
			delta.Add("Spec.PreferredBackupWindow", a.ko.Spec.PreferredBackupWindow, b.ko.Spec.PreferredBackupWindow)
		} else if a.ko.Spec.PreferredBackupWindow != nil && b.ko.Spec.PreferredBackupWindow != nil {
			if *a.ko.Spec.PreferredBackupWindow != *b.ko.Spec.PreferredBackupWindow {
				delta.Add("Spec.PreferredBackupWindow", a.ko.Spec.PreferredBackupWindow, b.ko.Spec.PreferredBackupWindow)
			}
		}

		if ackcompare.HasNilDifference(a.ko.Spec.DeletionProtection, b.ko.Spec.DeletionProtection) {
			delta.Add("Spec.DeletionProtection", a.ko.Spec.DeletionProtection, b.ko.Spec.DeletionProtection)
		} else if a.ko.Spec.DeletionProtection != nil && b.ko.Spec.DeletionProtection != nil {
			if *a.ko.Spec.DeletionProtection != *b.ko.Spec.DeletionProtection {
				delta.Add("Spec.DeletionProtection", a.ko.Spec.DeletionProtection, b.ko.Spec.DeletionProtection)
			}
		}
	}

}

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

func getCloudwatchLogExportsConfigDifferences(cloudwatchLogExportsConfigDesired []*string, cloudwatchLogExportsConfigLatest []*string) ([]*string, []*string) {
	logsTypesToEnable := []*string{}
	logsTypesToDisable := []*string{}
	desired := aws.ToStringSlice(cloudwatchLogExportsConfigDesired)
	latest := aws.ToStringSlice(cloudwatchLogExportsConfigLatest)

	for _, config := range cloudwatchLogExportsConfigDesired {
		if !slices.Contains(latest, *config) {
			logsTypesToEnable = append(logsTypesToEnable, config)
		}
	}
	for _, config := range cloudwatchLogExportsConfigLatest {
		if !slices.Contains(desired, *config) {
			logsTypesToDisable = append(logsTypesToDisable, config)
		}
	}
	return logsTypesToEnable, logsTypesToDisable
}

// manageCrossRegionBackupReplication handles enabling/disabling cross-region backup replication
func (rm *resourceManager) manageCrossRegionBackupReplication(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.manageCrossRegionBackupReplication")
	defer func(err error) { exit(err) }(err)

	// Check if replication state changed
	desiredEnabled := desired.ko.Spec.BackupCrossRegionReplication != nil &&
		*desired.ko.Spec.BackupCrossRegionReplication
	// Check status field because AWS doesn't populate the spec field
	latestEnabled := latest.ko.Status.DBInstanceAutomatedBackupsReplications != nil &&
		len(latest.ko.Status.DBInstanceAutomatedBackupsReplications) > 0

	// Enable replication
	if desiredEnabled && !latestEnabled {
		if desired.ko.Spec.BackupCrossRegionReplicationDestinationRegion == nil {
			return fmt.Errorf("BackupCrossRegionReplicationDestinationRegion is required when BackupCrossRegionReplication is true")
		}

		if latest.ko.Status.ACKResourceMetadata == nil || latest.ko.Status.ACKResourceMetadata.ARN == nil {
			return fmt.Errorf("DB instance ARN is required to enable cross-region backup replication")
		}

		// Check if automated backups are enabled on the source instance.
		// AWS requires backupRetentionPeriod > 0 before enabling cross-region replication.
		latestBackupRetentionPeriod := int64(-1)
		if latest.ko.Spec.BackupRetentionPeriod != nil {
			latestBackupRetentionPeriod = *latest.ko.Spec.BackupRetentionPeriod
		}
		desiredBackupRetentionPeriod := int64(-1)
		if desired.ko.Spec.BackupRetentionPeriod != nil {
			desiredBackupRetentionPeriod = *desired.ko.Spec.BackupRetentionPeriod
		}

		// Check for pending backup retention period changes
		pendingBackupRetentionPeriod := int64(-1)
		if latest.ko.Status.PendingModifiedValues != nil &&
			latest.ko.Status.PendingModifiedValues.BackupRetentionPeriod != nil {
			pendingBackupRetentionPeriod = *latest.ko.Status.PendingModifiedValues.BackupRetentionPeriod
		}

		// If backups status is unknown, wait for late-init/read to populate.
		if latestBackupRetentionPeriod < 0 {
			rlog.Info("BackupRetentionPeriod not yet observed; waiting before enabling cross-region backup replication")
			return ackrequeue.NeededAfter(
				errors.New("backup retention period not yet observed"),
				ackrequeue.DefaultRequeueAfterDuration,
			)
		}

		// If backups are not yet active, check if they're being enabled.
		// If backupRetentionPeriod is being modified (pending), wait for that to complete.
		// Otherwise, allow reconciliation to continue so Update() can call ModifyDBInstance.
		if latestBackupRetentionPeriod == 0 {
			if desiredBackupRetentionPeriod <= 0 {
				return fmt.Errorf("automated backups must be enabled (backupRetentionPeriod > 0) before enabling cross-region backup replication")
			}
			// If backups are pending (being modified), wait for that to complete
			if pendingBackupRetentionPeriod > 0 {
				rlog.Info("Waiting for pending backup retention period modification to complete before enabling cross-region backup replication",
					"pendingBackupRetentionPeriod", pendingBackupRetentionPeriod)
				return ackrequeue.NeededAfter(
					errors.New("backup retention period modification pending"),
					ackrequeue.DefaultRequeueAfterDuration,
				)
			}
			// Backups are not active and not pending - allow reconciliation to continue
			// so Update() can be called to enable backups via ModifyDBInstance.
			// Don't return an error here, as that would prevent Update() from being called.
			// Note: If BackupRetentionPeriod is in the delta, Update() will call ModifyDBInstance.
			// On the next reconciliation, PendingModifiedValues will be populated and we'll requeue.
			rlog.Info("Automated backups not yet active; allowing reconciliation to continue so ModifyDBInstance can enable backups")
			return nil
		}

		sourceARN := string(*latest.ko.Status.ACKResourceMetadata.ARN)
		input := &svcsdk.StartDBInstanceAutomatedBackupsReplicationInput{
			SourceDBInstanceArn: &sourceARN,
		}

		// Set retention period (default 7)
		retentionPeriod := int32(7)
		if desired.ko.Spec.BackupCrossRegionReplicationRetentionPeriod != nil {
			retentionPeriod = int32(*desired.ko.Spec.BackupCrossRegionReplicationRetentionPeriod)
		}
		input.BackupRetentionPeriod = &retentionPeriod

		// Set KMS key ID if specified
		if desired.ko.Spec.BackupCrossRegionReplicationKMSKeyID != nil {
			input.KmsKeyId = desired.ko.Spec.BackupCrossRegionReplicationKMSKeyID
		}

		// Create a client for the destination region
		// The AWS SDK uses the client's configured region to determine where the API call targets
		var apiClient *svcsdk.Client
		if desired.ko.Spec.BackupCrossRegionReplicationDestinationRegion != nil {
			destRegion := string(*desired.ko.Spec.BackupCrossRegionReplicationDestinationRegion)
			destConfig := rm.clientcfg.Copy()
			destConfig.Region = destRegion
			apiClient = svcsdk.NewFromConfig(destConfig)
		} else {
			apiClient = rm.sdkapi
		}

		// Start must be called in the destination region
		_, err := apiClient.StartDBInstanceAutomatedBackupsReplication(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "StartDBInstanceAutomatedBackupsReplication", err)
		if err != nil {
			// Handle idempotent case: if replication is already enabled, treat as success
			errMsg := err.Error()
			if strings.Contains(errMsg, "already replicating") {
				rlog.Info("Replication already enabled, treating as success")
				return nil
			}
			return err
		}
		return nil
	}

	// Disable replication
	if !desiredEnabled && latestEnabled {
		// Check if there are active replications
		if latest.ko.Status.DBInstanceAutomatedBackupsReplications == nil ||
			len(latest.ko.Status.DBInstanceAutomatedBackupsReplications) == 0 {
			rlog.Info("No active replication found to stop")
			return nil
		}

		if latest.ko.Status.ACKResourceMetadata == nil || latest.ko.Status.ACKResourceMetadata.ARN == nil {
			return fmt.Errorf("DB instance ARN is required to disable cross-region backup replication")
		}

		// Extract destination region from replication ARN in status
		destRegion := ""
		if len(latest.ko.Status.DBInstanceAutomatedBackupsReplications) > 0 &&
			latest.ko.Status.DBInstanceAutomatedBackupsReplications[0].DBInstanceAutomatedBackupsARN != nil {
			replicationARN := *latest.ko.Status.DBInstanceAutomatedBackupsReplications[0].DBInstanceAutomatedBackupsARN
			arnParts := strings.Split(replicationARN, ":")
			if len(arnParts) >= 4 {
				destRegion = arnParts[3]
			}
		}

		// Fallback to spec field if available
		if destRegion == "" && desired.ko.Spec.BackupCrossRegionReplicationDestinationRegion != nil {
			destRegion = string(*desired.ko.Spec.BackupCrossRegionReplicationDestinationRegion)
		}

		if destRegion == "" {
			return fmt.Errorf("could not determine destination region for stopping replication")
		}

		sourceARN := string(*latest.ko.Status.ACKResourceMetadata.ARN)
		input := &svcsdk.StopDBInstanceAutomatedBackupsReplicationInput{
			SourceDBInstanceArn: &sourceARN,
		}

		// Stop must be called from the destination region
		destConfig := rm.clientcfg.Copy()
		destConfig.Region = destRegion
		stopClient := svcsdk.NewFromConfig(destConfig)

		_, err := stopClient.StopDBInstanceAutomatedBackupsReplication(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "StopDBInstanceAutomatedBackupsReplication", err)
		if err != nil {
			// Handle idempotent case: if replication is not active, treat as success
			errMsg := err.Error()
			if strings.Contains(errMsg, "not replicating") {
				rlog.Info("Replication already stopped or not active, treating as success")
				return nil
			}
			return err
		}
		rlog.Info("Stopped cross-region backup replication")
		return nil
	}

	return nil
}
