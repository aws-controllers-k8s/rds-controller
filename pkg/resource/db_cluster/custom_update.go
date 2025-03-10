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

package db_cluster

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/rds"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

var r = regexp.MustCompile(`[0-9]*$`)

// customUpdate is required to fix
// https://github.com/aws-controllers-k8s/community/issues/917. The Input shape
// sent to ModifyDBCluster MUST have fields that are unchanged between desired
// and observed set to `nil`.
func (rm *resourceManager) customUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdate")
	defer exit(err)

	if clusterDeleting(latest) {
		msg := "DB cluster is currently being deleted"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitWhileDeleting
	}
	if clusterCreating(latest) {
		msg := "DB cluster is currently being created"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitUntilCanModify(latest)
	}
	if !clusterAvailable(latest) {
		msg := "DB cluster is not available for modification in '" +
			*latest.ko.Status.Status + "' status"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitUntilCanModify(latest)
	}
	if clusterHasTerminalStatus(latest) {
		msg := "DB cluster is in '" + *latest.ko.Status.Status + "' status"
		ackcondition.SetTerminal(desired, corev1.ConditionTrue, &msg, nil)
		ackcondition.SetSynced(desired, corev1.ConditionTrue, nil, nil)
		return desired, nil
	}
	if delta.DifferentAt("Spec.Tags") {
		if err = rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	} else if !delta.DifferentExcept("Spec.Tags") {
		// If the only difference between the desired and latest is in the
		// Spec.Tags field, we can skip the modify db cluster call.
		return desired, nil
	}

	input, err := rm.newCustomUpdateRequestPayload(ctx, desired, latest, delta)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.ModifyDBClusterOutput
	_ = resp
	resp, err = rm.sdkapi.ModifyDBCluster(ctx, input)

	rm.metrics.RecordAPICall("UPDATE", "ModifyDBCluster", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.DBCluster.ActivityStreamKinesisStreamName != nil {
		ko.Status.ActivityStreamKinesisStreamName = resp.DBCluster.ActivityStreamKinesisStreamName
	} else {
		ko.Status.ActivityStreamKinesisStreamName = nil
	}
	if resp.DBCluster.ActivityStreamKmsKeyId != nil {
		ko.Status.ActivityStreamKMSKeyID = resp.DBCluster.ActivityStreamKmsKeyId
	} else {
		ko.Status.ActivityStreamKMSKeyID = nil
	}
	if resp.DBCluster.ActivityStreamMode != "" {
		ko.Status.ActivityStreamMode = aws.String(string(resp.DBCluster.ActivityStreamMode))
	} else {
		ko.Status.ActivityStreamMode = nil
	}
	if resp.DBCluster.ActivityStreamStatus != "" {
		ko.Status.ActivityStreamStatus = aws.String(string(resp.DBCluster.ActivityStreamStatus))
	} else {
		ko.Status.ActivityStreamStatus = nil
	}
	if resp.DBCluster.AllocatedStorage != nil {
		ko.Spec.AllocatedStorage = aws.Int64(int64(*resp.DBCluster.AllocatedStorage))
	} else {
		ko.Spec.AllocatedStorage = nil
	}
	if resp.DBCluster.AssociatedRoles != nil {
		f5 := []*svcapitypes.DBClusterRole{}
		for _, f5iter := range resp.DBCluster.AssociatedRoles {
			f5elem := &svcapitypes.DBClusterRole{}
			if f5iter.FeatureName != nil {
				f5elem.FeatureName = f5iter.FeatureName
			}
			if f5iter.RoleArn != nil {
				f5elem.RoleARN = f5iter.RoleArn
			}
			if f5iter.Status != nil {
				f5elem.Status = f5iter.Status
			}
			f5 = append(f5, f5elem)
		}
		ko.Status.AssociatedRoles = f5
	} else {
		ko.Status.AssociatedRoles = nil
	}
	if resp.DBCluster.AvailabilityZones != nil {
		ko.Spec.AvailabilityZones = aws.StringSlice(resp.DBCluster.AvailabilityZones)
	} else {
		ko.Spec.AvailabilityZones = nil
	}
	if resp.DBCluster.BacktrackConsumedChangeRecords != nil {
		ko.Status.BacktrackConsumedChangeRecords = resp.DBCluster.BacktrackConsumedChangeRecords
	} else {
		ko.Status.BacktrackConsumedChangeRecords = nil
	}
	if resp.DBCluster.BacktrackWindow != nil {
		ko.Spec.BacktrackWindow = resp.DBCluster.BacktrackWindow
	} else {
		ko.Spec.BacktrackWindow = nil
	}
	if resp.DBCluster.BackupRetentionPeriod != nil {
		ko.Spec.BackupRetentionPeriod = aws.Int64(int64(*resp.DBCluster.BackupRetentionPeriod))
	} else {
		ko.Spec.BackupRetentionPeriod = nil
	}
	if resp.DBCluster.Capacity != nil {
		ko.Status.Capacity = aws.Int64(int64(*resp.DBCluster.Capacity))
	} else {
		ko.Status.Capacity = nil
	}
	if resp.DBCluster.CharacterSetName != nil {
		ko.Spec.CharacterSetName = resp.DBCluster.CharacterSetName
	} else {
		ko.Spec.CharacterSetName = nil
	}
	if resp.DBCluster.CloneGroupId != nil {
		ko.Status.CloneGroupID = resp.DBCluster.CloneGroupId
	} else {
		ko.Status.CloneGroupID = nil
	}
	if resp.DBCluster.ClusterCreateTime != nil {
		ko.Status.ClusterCreateTime = &metav1.Time{Time: *resp.DBCluster.ClusterCreateTime}
	} else {
		ko.Status.ClusterCreateTime = nil
	}
	if resp.DBCluster.CopyTagsToSnapshot != nil {
		ko.Spec.CopyTagsToSnapshot = resp.DBCluster.CopyTagsToSnapshot
	} else {
		ko.Spec.CopyTagsToSnapshot = nil
	}
	if resp.DBCluster.CrossAccountClone != nil {
		ko.Status.CrossAccountClone = resp.DBCluster.CrossAccountClone
	} else {
		ko.Status.CrossAccountClone = nil
	}
	if resp.DBCluster.CustomEndpoints != nil {
		ko.Status.CustomEndpoints = aws.StringSlice(resp.DBCluster.CustomEndpoints)
	} else {
		ko.Status.CustomEndpoints = nil
	}
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.DBCluster.DBClusterArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.DBCluster.DBClusterArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.DBCluster.DBClusterIdentifier != nil {
		ko.Spec.DBClusterIdentifier = resp.DBCluster.DBClusterIdentifier
	} else {
		ko.Spec.DBClusterIdentifier = nil
	}
	if resp.DBCluster.DBClusterMembers != nil {
		f19 := []*svcapitypes.DBClusterMember{}
		for _, f19iter := range resp.DBCluster.DBClusterMembers {
			f19elem := &svcapitypes.DBClusterMember{}
			if f19iter.DBClusterParameterGroupStatus != nil {
				f19elem.DBClusterParameterGroupStatus = f19iter.DBClusterParameterGroupStatus
			}
			if f19iter.DBInstanceIdentifier != nil {
				f19elem.DBInstanceIdentifier = f19iter.DBInstanceIdentifier
			}
			if f19iter.IsClusterWriter != nil {
				f19elem.IsClusterWriter = f19iter.IsClusterWriter
			}
			if f19iter.PromotionTier != nil {
				f19elem.PromotionTier = aws.Int64(int64(*f19iter.PromotionTier))
			}
			f19 = append(f19, f19elem)
		}
		ko.Status.DBClusterMembers = f19
	} else {
		ko.Status.DBClusterMembers = nil
	}
	if resp.DBCluster.DBClusterOptionGroupMemberships != nil {
		f20 := []*svcapitypes.DBClusterOptionGroupStatus{}
		for _, f20iter := range resp.DBCluster.DBClusterOptionGroupMemberships {
			f20elem := &svcapitypes.DBClusterOptionGroupStatus{}
			if f20iter.DBClusterOptionGroupName != nil {
				f20elem.DBClusterOptionGroupName = f20iter.DBClusterOptionGroupName
			}
			if f20iter.Status != nil {
				f20elem.Status = f20iter.Status
			}
			f20 = append(f20, f20elem)
		}
		ko.Status.DBClusterOptionGroupMemberships = f20
	} else {
		ko.Status.DBClusterOptionGroupMemberships = nil
	}
	if resp.DBCluster.DBClusterParameterGroup != nil {
		ko.Status.DBClusterParameterGroup = resp.DBCluster.DBClusterParameterGroup
	} else {
		ko.Status.DBClusterParameterGroup = nil
	}
	if resp.DBCluster.DBSubnetGroup != nil {
		ko.Status.DBSubnetGroup = resp.DBCluster.DBSubnetGroup
	} else {
		ko.Status.DBSubnetGroup = nil
	}
	if resp.DBCluster.DatabaseName != nil {
		ko.Spec.DatabaseName = resp.DBCluster.DatabaseName
	} else {
		ko.Spec.DatabaseName = nil
	}
	if resp.DBCluster.DbClusterResourceId != nil {
		ko.Status.DBClusterResourceID = resp.DBCluster.DbClusterResourceId
	} else {
		ko.Status.DBClusterResourceID = nil
	}
	if resp.DBCluster.DeletionProtection != nil {
		ko.Spec.DeletionProtection = resp.DBCluster.DeletionProtection
	} else {
		ko.Spec.DeletionProtection = nil
	}
	if resp.DBCluster.DomainMemberships != nil {
		f26 := []*svcapitypes.DomainMembership{}
		for _, f26iter := range resp.DBCluster.DomainMemberships {
			f26elem := &svcapitypes.DomainMembership{}
			if f26iter.Domain != nil {
				f26elem.Domain = f26iter.Domain
			}
			if f26iter.FQDN != nil {
				f26elem.FQDN = f26iter.FQDN
			}
			if f26iter.IAMRoleName != nil {
				f26elem.IAMRoleName = f26iter.IAMRoleName
			}
			if f26iter.Status != nil {
				f26elem.Status = f26iter.Status
			}
			f26 = append(f26, f26elem)
		}
		ko.Status.DomainMemberships = f26
	} else {
		ko.Status.DomainMemberships = nil
	}
	if resp.DBCluster.EarliestBacktrackTime != nil {
		ko.Status.EarliestBacktrackTime = &metav1.Time{*resp.DBCluster.EarliestBacktrackTime}
	} else {
		ko.Status.EarliestBacktrackTime = nil
	}
	if resp.DBCluster.EarliestRestorableTime != nil {
		ko.Status.EarliestRestorableTime = &metav1.Time{*resp.DBCluster.EarliestRestorableTime}
	} else {
		ko.Status.EarliestRestorableTime = nil
	}
	if resp.DBCluster.EnabledCloudwatchLogsExports != nil {
		ko.Status.EnabledCloudwatchLogsExports = aws.StringSlice(resp.DBCluster.EnabledCloudwatchLogsExports)
	} else {
		ko.Status.EnabledCloudwatchLogsExports = nil
	}
	if resp.DBCluster.Endpoint != nil {
		ko.Status.Endpoint = resp.DBCluster.Endpoint
	} else {
		ko.Status.Endpoint = nil
	}
	if resp.DBCluster.Engine != nil {
		ko.Spec.Engine = resp.DBCluster.Engine
	} else {
		ko.Spec.Engine = nil
	}
	if resp.DBCluster.EngineMode != nil {
		ko.Spec.EngineMode = resp.DBCluster.EngineMode
	} else {
		ko.Spec.EngineMode = nil
	}
	if resp.DBCluster.EngineVersion != nil {
		ko.Spec.EngineVersion = resp.DBCluster.EngineVersion
	} else {
		ko.Spec.EngineVersion = nil
	}
	if resp.DBCluster.GlobalWriteForwardingRequested != nil {
		ko.Status.GlobalWriteForwardingRequested = resp.DBCluster.GlobalWriteForwardingRequested
	} else {
		ko.Status.GlobalWriteForwardingRequested = nil
	}
	if resp.DBCluster.GlobalWriteForwardingStatus != "" {
		ko.Status.GlobalWriteForwardingStatus = aws.String(string(resp.DBCluster.GlobalWriteForwardingStatus))
	} else {
		ko.Status.GlobalWriteForwardingStatus = nil
	}
	if resp.DBCluster.HostedZoneId != nil {
		ko.Status.HostedZoneID = resp.DBCluster.HostedZoneId
	} else {
		ko.Status.HostedZoneID = nil
	}
	if resp.DBCluster.HttpEndpointEnabled != nil {
		ko.Status.HTTPEndpointEnabled = resp.DBCluster.HttpEndpointEnabled
	} else {
		ko.Status.HTTPEndpointEnabled = nil
	}
	if resp.DBCluster.IAMDatabaseAuthenticationEnabled != nil {
		ko.Status.IAMDatabaseAuthenticationEnabled = resp.DBCluster.IAMDatabaseAuthenticationEnabled
	} else {
		ko.Status.IAMDatabaseAuthenticationEnabled = nil
	}
	if resp.DBCluster.KmsKeyId != nil {
		ko.Spec.KMSKeyID = resp.DBCluster.KmsKeyId
	} else {
		ko.Spec.KMSKeyID = nil
	}
	if resp.DBCluster.LatestRestorableTime != nil {
		ko.Status.LatestRestorableTime = &metav1.Time{*resp.DBCluster.LatestRestorableTime}
	} else {
		ko.Status.LatestRestorableTime = nil
	}
	if resp.DBCluster.MasterUsername != nil {
		ko.Spec.MasterUsername = resp.DBCluster.MasterUsername
	} else {
		ko.Spec.MasterUsername = nil
	}
	if resp.DBCluster.MultiAZ != nil {
		ko.Status.MultiAZ = resp.DBCluster.MultiAZ
	} else {
		ko.Status.MultiAZ = nil
	}
	if resp.DBCluster.PendingModifiedValues != nil {
		f43 := &svcapitypes.ClusterPendingModifiedValues{}
		if resp.DBCluster.PendingModifiedValues.DBClusterIdentifier != nil {
			f43.DBClusterIdentifier = resp.DBCluster.PendingModifiedValues.DBClusterIdentifier
		}
		if resp.DBCluster.PendingModifiedValues.EngineVersion != nil {
			f43.EngineVersion = resp.DBCluster.PendingModifiedValues.EngineVersion
		}
		if resp.DBCluster.PendingModifiedValues.IAMDatabaseAuthenticationEnabled != nil {
			f43.IAMDatabaseAuthenticationEnabled = resp.DBCluster.PendingModifiedValues.IAMDatabaseAuthenticationEnabled
		}
		if resp.DBCluster.PendingModifiedValues.MasterUserPassword != nil {
			f43.MasterUserPassword = resp.DBCluster.PendingModifiedValues.MasterUserPassword
		}
		if resp.DBCluster.PendingModifiedValues.PendingCloudwatchLogsExports != nil {
			f43f4 := &svcapitypes.PendingCloudwatchLogsExports{}
			if resp.DBCluster.PendingModifiedValues.PendingCloudwatchLogsExports.LogTypesToDisable != nil {
				f43f4.LogTypesToDisable = aws.StringSlice(resp.DBCluster.PendingModifiedValues.PendingCloudwatchLogsExports.LogTypesToDisable)
			}
			if resp.DBCluster.PendingModifiedValues.PendingCloudwatchLogsExports.LogTypesToEnable != nil {
				f43f4.LogTypesToEnable = aws.StringSlice(resp.DBCluster.PendingModifiedValues.PendingCloudwatchLogsExports.LogTypesToEnable)
			}
			f43.PendingCloudwatchLogsExports = f43f4
		}
		ko.Status.PendingModifiedValues = f43
	} else {
		ko.Status.PendingModifiedValues = nil
	}
	if resp.DBCluster.PercentProgress != nil {
		ko.Status.PercentProgress = resp.DBCluster.PercentProgress
	} else {
		ko.Status.PercentProgress = nil
	}
	if resp.DBCluster.Port != nil {
		ko.Spec.Port = aws.Int64(int64(*resp.DBCluster.Port))
	} else {
		ko.Spec.Port = nil
	}
	if resp.DBCluster.PreferredBackupWindow != nil {
		ko.Spec.PreferredBackupWindow = resp.DBCluster.PreferredBackupWindow
	} else {
		ko.Spec.PreferredBackupWindow = nil
	}
	if resp.DBCluster.PreferredMaintenanceWindow != nil {
		ko.Spec.PreferredMaintenanceWindow = resp.DBCluster.PreferredMaintenanceWindow
	} else {
		ko.Spec.PreferredMaintenanceWindow = nil
	}
	if resp.DBCluster.ReadReplicaIdentifiers != nil {
		ko.Status.ReadReplicaIdentifiers = aws.StringSlice(resp.DBCluster.ReadReplicaIdentifiers)
	} else {
		ko.Status.ReadReplicaIdentifiers = nil
	}
	if resp.DBCluster.ReaderEndpoint != nil {
		ko.Status.ReaderEndpoint = resp.DBCluster.ReaderEndpoint
	} else {
		ko.Status.ReaderEndpoint = nil
	}
	if resp.DBCluster.ReplicationSourceIdentifier != nil {
		ko.Spec.ReplicationSourceIdentifier = resp.DBCluster.ReplicationSourceIdentifier
	} else {
		ko.Spec.ReplicationSourceIdentifier = nil
	}
	if resp.DBCluster.Status != nil {
		ko.Status.Status = resp.DBCluster.Status
	} else {
		ko.Status.Status = nil
	}
	if resp.DBCluster.StorageEncrypted != nil {
		ko.Spec.StorageEncrypted = resp.DBCluster.StorageEncrypted
	} else {
		ko.Spec.StorageEncrypted = nil
	}
	if resp.DBCluster.TagList != nil {
		f54 := []*svcapitypes.Tag{}
		for _, f54iter := range resp.DBCluster.TagList {
			f54elem := &svcapitypes.Tag{}
			if f54iter.Key != nil {
				f54elem.Key = f54iter.Key
			}
			if f54iter.Value != nil {
				f54elem.Value = f54iter.Value
			}
			f54 = append(f54, f54elem)
		}
		ko.Status.TagList = f54
	} else {
		ko.Status.TagList = nil
	}
	if resp.DBCluster.VpcSecurityGroups != nil {
		f55 := []*svcapitypes.VPCSecurityGroupMembership{}
		for _, f55iter := range resp.DBCluster.VpcSecurityGroups {
			f55elem := &svcapitypes.VPCSecurityGroupMembership{}
			if f55iter.Status != nil {
				f55elem.Status = f55iter.Status
			}
			if f55iter.VpcSecurityGroupId != nil {
				f55elem.VPCSecurityGroupID = f55iter.VpcSecurityGroupId
			}
			f55 = append(f55, f55elem)
		}
		ko.Status.VPCSecurityGroups = f55
	} else {
		ko.Status.VPCSecurityGroups = nil
	}
	rm.setStatusDefaults(ko)
	// When ModifyDBInstance API is successful, it asynchronously
	// updates the DBInstanceStatus. Requeue to find the current
	// DBInstance status and set Synced condition accordingly
	r := &resource{ko}
	if err == nil {
		// set the last-applied-secret-reference annotation on the DB instance
		// resource.
		setLastAppliedSecretReferenceAnnotation(r)
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(r, corev1.ConditionFalse, nil, nil)
	}
	return r, nil
}

// newCustomUpdateRequestPayload returns an SDK-specific struct for the HTTP
// request payload of the Update API call for the resource. It is different
// from the normal newUpdateRequestsPayload in that in addition to checking for
// nil-ness of the Spec fields, it also checks to see if the delta between
// desired and observed contains a diff for the specific field. This is
// required in order to fix
// https://github.com/aws-controllers-k8s/community/issues/917
func (rm *resourceManager) newCustomUpdateRequestPayload(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (*svcsdk.ModifyDBClusterInput, error) {
	res := &svcsdk.ModifyDBClusterInput{}

	res.ApplyImmediately = aws.Bool(true)
	res.AllowMajorVersionUpgrade = aws.Bool(true)
	if desired.ko.Spec.BacktrackWindow != nil && delta.DifferentAt("Spec.BacktrackWindow") {
		res.BacktrackWindow = desired.ko.Spec.BacktrackWindow
	}
	if desired.ko.Spec.BackupRetentionPeriod != nil && delta.DifferentAt("Spec.BackupRetentionPeriod") {
		if *desired.ko.Spec.BackupRetentionPeriod > math.MaxInt32 {
			return nil, fmt.Errorf("error: BackupRetentionPeriod should be int32")
		}
		res.BackupRetentionPeriod = aws.Int32(int32(*desired.ko.Spec.BackupRetentionPeriod))
	}
	if desired.ko.Spec.CopyTagsToSnapshot != nil && delta.DifferentAt("Spec.CopyTagsToSnapshot") {
		res.CopyTagsToSnapshot = desired.ko.Spec.CopyTagsToSnapshot
	}
	// NOTE(jaypipes): This is a required field in the input shape. If not set,
	// we get back a cryptic error message "1 Validation error(s) found."
	if desired.ko.Spec.DBClusterIdentifier != nil {
		res.DBClusterIdentifier = desired.ko.Spec.DBClusterIdentifier
	}
	if desired.ko.Spec.DBClusterParameterGroupName != nil && delta.DifferentAt("Spec.DBClusterParameterGroupName") {
		res.DBClusterParameterGroupName = desired.ko.Spec.DBClusterParameterGroupName
	}
	if desired.ko.Spec.DeletionProtection != nil && delta.DifferentAt("Spec.DeletionProtection") {
		res.DeletionProtection = desired.ko.Spec.DeletionProtection
	}
	if desired.ko.Spec.Domain != nil && delta.DifferentAt("Spec.Domain") {
		res.Domain = desired.ko.Spec.Domain
	}
	if desired.ko.Spec.DomainIAMRoleName != nil && delta.DifferentAt("Spec.DomainIAMRoleName") {
		res.DomainIAMRoleName = desired.ko.Spec.DomainIAMRoleName
	}
	if desired.ko.Spec.EnableGlobalWriteForwarding != nil && delta.DifferentAt("Spec.EnableGlobalWriteForwarding") {
		res.EnableGlobalWriteForwarding = desired.ko.Spec.EnableGlobalWriteForwarding
	}
	if desired.ko.Spec.EnableHTTPEndpoint != nil && delta.DifferentAt("Spec.EnableHTTPEndpoint") {
		res.EnableHttpEndpoint = desired.ko.Spec.EnableHTTPEndpoint
	}
	if desired.ko.Spec.EnableIAMDatabaseAuthentication != nil && delta.DifferentAt("Spec.EnableIAMDatabaseAuthentication") {
		res.EnableIAMDatabaseAuthentication = desired.ko.Spec.EnableIAMDatabaseAuthentication
	}
	if desired.ko.Spec.EngineVersion != nil && delta.DifferentAt("Spec.EngineVersion") {
		autoMinorVersionUpgrade := true
		if desired.ko.Spec.AutoMinorVersionUpgrade != nil {
			autoMinorVersionUpgrade = *desired.ko.Spec.AutoMinorVersionUpgrade
		}
		if requireEngineVersionUpdate(desired.ko.Spec.EngineVersion, latest.ko.Spec.EngineVersion, autoMinorVersionUpgrade) {
			res.EngineVersion = desired.ko.Spec.EngineVersion
		}
	}
	if desired.ko.Spec.MasterUserPassword != nil && delta.DifferentAt("Spec.MasterUserPassword") {
		tmpSecret, err := rm.rr.SecretValueFromReference(ctx, desired.ko.Spec.MasterUserPassword)
		if err != nil {
			return nil, err
		}
		if tmpSecret != "" {
			res.MasterUserPassword = &tmpSecret
		}
	}
	if desired.ko.Spec.OptionGroupName != nil && delta.DifferentAt("Spec.OptionGroupName") {
		res.OptionGroupName = desired.ko.Spec.OptionGroupName
	}
	if desired.ko.Spec.StorageType != nil && delta.DifferentAt("Spec.StorageType") {
		res.StorageType = desired.ko.Spec.StorageType
	}
	if desired.ko.Spec.Port != nil && delta.DifferentAt("Spec.Port") {
		res.Port = aws.Int32(int32(*desired.ko.Spec.Port))
	}
	if desired.ko.Spec.PreferredBackupWindow != nil && delta.DifferentAt("Spec.PreferredBackupkWindow") {
		res.PreferredBackupWindow = desired.ko.Spec.PreferredBackupWindow
	}
	if desired.ko.Spec.PreferredMaintenanceWindow != nil && delta.DifferentAt("Spec.PreferredMaintenanceWindow") {
		res.PreferredMaintenanceWindow = desired.ko.Spec.PreferredMaintenanceWindow
	}
	if desired.ko.Spec.ScalingConfiguration != nil && delta.DifferentAt("Spec.ScalingConfiguration") {
		f22 := &svcsdktypes.ScalingConfiguration{}
		if desired.ko.Spec.ScalingConfiguration.AutoPause != nil && delta.DifferentAt("Spec.ScalingConfiguration.AutoPause") {
			f22.AutoPause = desired.ko.Spec.ScalingConfiguration.AutoPause
		}
		if desired.ko.Spec.ScalingConfiguration.MaxCapacity != nil && delta.DifferentAt("Spec.ScalingConfiguration.MaxCapacity") {
			f22.MaxCapacity = aws.Int32(int32(*desired.ko.Spec.ScalingConfiguration.MaxCapacity))
		}
		if desired.ko.Spec.ScalingConfiguration.MinCapacity != nil && delta.DifferentAt("Spec.ScalingConfiguration.MinCapacity") {
			f22.MinCapacity = aws.Int32(int32(*desired.ko.Spec.ScalingConfiguration.MinCapacity))
		}
		if desired.ko.Spec.ScalingConfiguration.SecondsUntilAutoPause != nil && delta.DifferentAt("Spec.ScalingConfiguration.SecondsUntilAutoPause") {
			f22.SecondsUntilAutoPause = aws.Int32(int32(*desired.ko.Spec.ScalingConfiguration.SecondsUntilAutoPause))
		}
		if desired.ko.Spec.ScalingConfiguration.TimeoutAction != nil && delta.DifferentAt("Spec.ScalingConfiguration.TimeoutAction") {
			f22.TimeoutAction = desired.ko.Spec.ScalingConfiguration.TimeoutAction
		}
		res.ScalingConfiguration = (f22)
	}
	if desired.ko.Spec.VPCSecurityGroupIDs != nil && delta.DifferentAt("Spec.VPCSecurityGroupIDs") {
		res.VpcSecurityGroupIds = aws.ToStringSlice(desired.ko.Spec.VPCSecurityGroupIDs)
	}
	// For ServerlessV2ScalingConfiguration, MaxCapacity and MinCapacity,  both need appear in modify call to get ServerlessV2ScalingConfiguration modified
	if desired.ko.Spec.ServerlessV2ScalingConfiguration != nil && delta.DifferentAt("Spec.ServerlessV2ScalingConfiguration") {
		f23 := &svcsdktypes.ServerlessV2ScalingConfiguration{}
		if delta.DifferentAt("Spec.ServerlessV2ScalingConfiguration.MaxCapacity") || delta.DifferentAt("Spec.ServerlessV2ScalingConfiguration.MinCapacity") {
			if desired.ko.Spec.ServerlessV2ScalingConfiguration.MaxCapacity != nil {
				f23.MaxCapacity = desired.ko.Spec.ServerlessV2ScalingConfiguration.MaxCapacity
			}
			if desired.ko.Spec.ServerlessV2ScalingConfiguration.MaxCapacity != nil {
				f23.MinCapacity = desired.ko.Spec.ServerlessV2ScalingConfiguration.MinCapacity
			}
		}
		res.ServerlessV2ScalingConfiguration = f23
	}

	if delta.DifferentAt("Spec.EnableCloudwatchLogsExports") {
		cloudwatchLogExportsConfigDesired := desired.ko.Spec.EnableCloudwatchLogsExports
		//Latest log types config
		cloudwatchLogExportsConfigLatest := latest.ko.Spec.EnableCloudwatchLogsExports
		logsTypesToEnable, logsTypesToDisable := getCloudwatchLogExportsConfigDifferences(cloudwatchLogExportsConfigDesired, cloudwatchLogExportsConfigLatest)
		f24 := &svcsdktypes.CloudwatchLogsExportConfiguration{
			EnableLogTypes:  aws.ToStringSlice(logsTypesToEnable),
			DisableLogTypes: aws.ToStringSlice(logsTypesToDisable),
		}
		res.CloudwatchLogsExportConfiguration = f24
	}
	return res, nil
}

func getCloudwatchLogExportsConfigDifferences(cloudwatchLogExportsConfigDesired []*string, cloudwatchLogExportsConfigLatest []*string) ([]*string, []*string) {
	logsTypesToEnable := []*string{}
	logsTypesToDisable := []*string{}

	for _, config := range cloudwatchLogExportsConfigDesired {
		if !slices.Contains(cloudwatchLogExportsConfigLatest, config) {
			logsTypesToEnable = append(logsTypesToEnable, config)
		}
	}
	for _, config := range cloudwatchLogExportsConfigLatest {
		if !slices.Contains(cloudwatchLogExportsConfigDesired, config) {
			logsTypesToDisable = append(logsTypesToDisable, config)
		}
	}
	return logsTypesToEnable, logsTypesToDisable
}

func requireEngineVersionUpdate(desiredEngineVersion *string, latestEngineVersion *string, autoMinorVersionUpgrade bool) bool {
	desiredMajorEngineVersion := r.ReplaceAllString(*desiredEngineVersion, "${1}")
	latestMajorEngineVersion := r.ReplaceAllString(*latestEngineVersion, "${1}")
	return !autoMinorVersionUpgrade || desiredMajorEngineVersion != latestMajorEngineVersion
}
