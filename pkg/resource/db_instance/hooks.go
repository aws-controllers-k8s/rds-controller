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
	"errors"
	"fmt"

	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
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

	customWaitAtferUpdate = ackrequeue.NeededAfter(
		errors.New("Requeueing resource after successful update; status fields "+
			"retrieved asynchronously"),
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

// funtion to check if DBSnapshotIdentifier is not null,
// it will create new payload and call restoreDbInstanceFromDbSnapshot API
func (rm *resourceManager) restoreDbInstanceFromDbSnapshot(
	ctx context.Context,
	r *resource,
) (*resource, error) {
	if r.ko.Spec.DBSnapshotIdentifier == nil {
		return nil, nil
	}

	input, err := rm.restoreDbInstanceFromDbSnapshotPayload(r)
	if err != nil {
		return nil, err
	}

	resp, respErr := rm.sdkapi.RestoreDbInstanceFromDbSnapshot(input)

	rm.metrics.RecordAPICall("CREATE", "RestoreDbInstanceFromDbSnapshot", respErr)
	if respErr != nil {
		return nil, respErr
	}

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := r.ko.DeepCopy()

	if resp.DBInstance.ActivityStreamEngineNativeAuditFieldsIncluded != nil {
		ko.Status.ActivityStreamEngineNativeAuditFieldsIncluded = resp.DBInstance.ActivityStreamEngineNativeAuditFieldsIncluded
	} else {
		ko.Status.ActivityStreamEngineNativeAuditFieldsIncluded = nil
	}
	if resp.DBInstance.ActivityStreamKinesisStreamName != nil {
		ko.Status.ActivityStreamKinesisStreamName = resp.DBInstance.ActivityStreamKinesisStreamName
	} else {
		ko.Status.ActivityStreamKinesisStreamName = nil
	}
	if resp.DBInstance.ActivityStreamKmsKeyId != nil {
		ko.Status.ActivityStreamKMSKeyID = resp.DBInstance.ActivityStreamKmsKeyId
	} else {
		ko.Status.ActivityStreamKMSKeyID = nil
	}
	if resp.DBInstance.ActivityStreamMode != nil {
		ko.Status.ActivityStreamMode = resp.DBInstance.ActivityStreamMode
	} else {
		ko.Status.ActivityStreamMode = nil
	}
	if resp.DBInstance.ActivityStreamStatus != nil {
		ko.Status.ActivityStreamStatus = resp.DBInstance.ActivityStreamStatus
	} else {
		ko.Status.ActivityStreamStatus = nil
	}
	if resp.DBInstance.AllocatedStorage != nil {
		ko.Spec.AllocatedStorage = resp.DBInstance.AllocatedStorage
	} else {
		ko.Spec.AllocatedStorage = nil
	}
	if resp.DBInstance.AssociatedRoles != nil {
		f6 := []*svcapitypes.DBInstanceRole{}
		for _, f6iter := range resp.DBInstance.AssociatedRoles {
			f6elem := &svcapitypes.DBInstanceRole{}
			if f6iter.FeatureName != nil {
				f6elem.FeatureName = f6iter.FeatureName
			}
			if f6iter.RoleArn != nil {
				f6elem.RoleARN = f6iter.RoleArn
			}
			if f6iter.Status != nil {
				f6elem.Status = f6iter.Status
			}
			f6 = append(f6, f6elem)
		}
		ko.Status.AssociatedRoles = f6
	} else {
		ko.Status.AssociatedRoles = nil
	}
	if resp.DBInstance.AutoMinorVersionUpgrade != nil {
		ko.Spec.AutoMinorVersionUpgrade = resp.DBInstance.AutoMinorVersionUpgrade
	} else {
		ko.Spec.AutoMinorVersionUpgrade = nil
	}
	if resp.DBInstance.AutomaticRestartTime != nil {
		ko.Status.AutomaticRestartTime = &metav1.Time{*resp.DBInstance.AutomaticRestartTime}
	} else {
		ko.Status.AutomaticRestartTime = nil
	}
	if resp.DBInstance.AutomationMode != nil {
		ko.Status.AutomationMode = resp.DBInstance.AutomationMode
	} else {
		ko.Status.AutomationMode = nil
	}
	if resp.DBInstance.AvailabilityZone != nil {
		ko.Spec.AvailabilityZone = resp.DBInstance.AvailabilityZone
	} else {
		ko.Spec.AvailabilityZone = nil
	}
	if resp.DBInstance.AwsBackupRecoveryPointArn != nil {
		ko.Status.AWSBackupRecoveryPointARN = resp.DBInstance.AwsBackupRecoveryPointArn
	} else {
		ko.Status.AWSBackupRecoveryPointARN = nil
	}
	if resp.DBInstance.BackupRetentionPeriod != nil {
		ko.Spec.BackupRetentionPeriod = resp.DBInstance.BackupRetentionPeriod
	} else {
		ko.Spec.BackupRetentionPeriod = nil
	}
	if resp.DBInstance.CACertificateIdentifier != nil {
		ko.Status.CACertificateIdentifier = resp.DBInstance.CACertificateIdentifier
	} else {
		ko.Status.CACertificateIdentifier = nil
	}
	if resp.DBInstance.CharacterSetName != nil {
		ko.Spec.CharacterSetName = resp.DBInstance.CharacterSetName
	} else {
		ko.Spec.CharacterSetName = nil
	}
	if resp.DBInstance.CopyTagsToSnapshot != nil {
		ko.Spec.CopyTagsToSnapshot = resp.DBInstance.CopyTagsToSnapshot
	} else {
		ko.Spec.CopyTagsToSnapshot = nil
	}
	if resp.DBInstance.CustomIamInstanceProfile != nil {
		ko.Spec.CustomIAMInstanceProfile = resp.DBInstance.CustomIamInstanceProfile
	} else {
		ko.Spec.CustomIAMInstanceProfile = nil
	}
	if resp.DBInstance.CustomerOwnedIpEnabled != nil {
		ko.Status.CustomerOwnedIPEnabled = resp.DBInstance.CustomerOwnedIpEnabled
	} else {
		ko.Status.CustomerOwnedIPEnabled = nil
	}
	if resp.DBInstance.DBClusterIdentifier != nil {
		ko.Spec.DBClusterIdentifier = resp.DBInstance.DBClusterIdentifier
	} else {
		ko.Spec.DBClusterIdentifier = nil
	}
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.DBInstance.DBInstanceArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.DBInstance.DBInstanceArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.DBInstance.DBInstanceAutomatedBackupsReplications != nil {
		f20 := []*svcapitypes.DBInstanceAutomatedBackupsReplication{}
		for _, f20iter := range resp.DBInstance.DBInstanceAutomatedBackupsReplications {
			f20elem := &svcapitypes.DBInstanceAutomatedBackupsReplication{}
			if f20iter.DBInstanceAutomatedBackupsArn != nil {
				f20elem.DBInstanceAutomatedBackupsARN = f20iter.DBInstanceAutomatedBackupsArn
			}
			f20 = append(f20, f20elem)
		}
		ko.Status.DBInstanceAutomatedBackupsReplications = f20
	} else {
		ko.Status.DBInstanceAutomatedBackupsReplications = nil
	}
	if resp.DBInstance.DBInstanceClass != nil {
		ko.Spec.DBInstanceClass = resp.DBInstance.DBInstanceClass
	} else {
		ko.Spec.DBInstanceClass = nil
	}
	if resp.DBInstance.DBInstanceIdentifier != nil {
		ko.Spec.DBInstanceIdentifier = resp.DBInstance.DBInstanceIdentifier
	} else {
		ko.Spec.DBInstanceIdentifier = nil
	}
	if resp.DBInstance.DBInstanceStatus != nil {
		ko.Status.DBInstanceStatus = resp.DBInstance.DBInstanceStatus
	} else {
		ko.Status.DBInstanceStatus = nil
	}
	if resp.DBInstance.DBName != nil {
		ko.Spec.DBName = resp.DBInstance.DBName
	} else {
		ko.Spec.DBName = nil
	}
	if resp.DBInstance.DBParameterGroups != nil {
		f25 := []*svcapitypes.DBParameterGroupStatus_SDK{}
		for _, f25iter := range resp.DBInstance.DBParameterGroups {
			f25elem := &svcapitypes.DBParameterGroupStatus_SDK{}
			if f25iter.DBParameterGroupName != nil {
				f25elem.DBParameterGroupName = f25iter.DBParameterGroupName
			}
			if f25iter.ParameterApplyStatus != nil {
				f25elem.ParameterApplyStatus = f25iter.ParameterApplyStatus
			}
			f25 = append(f25, f25elem)
		}
		ko.Status.DBParameterGroups = f25
	} else {
		ko.Status.DBParameterGroups = nil
	}
	if resp.DBInstance.DBSecurityGroups != nil {
		f26 := []*string{}
		for _, f26iter := range resp.DBInstance.DBSecurityGroups {
			var f26elem string
			f26elem = *f26iter.DBSecurityGroupName
			f26 = append(f26, &f26elem)
		}
		ko.Spec.DBSecurityGroups = f26
	} else {
		ko.Spec.DBSecurityGroups = nil
	}
	if resp.DBInstance.DBSubnetGroup != nil {
		f27 := &svcapitypes.DBSubnetGroup_SDK{}
		if resp.DBInstance.DBSubnetGroup.DBSubnetGroupArn != nil {
			f27.DBSubnetGroupARN = resp.DBInstance.DBSubnetGroup.DBSubnetGroupArn
		}
		if resp.DBInstance.DBSubnetGroup.DBSubnetGroupDescription != nil {
			f27.DBSubnetGroupDescription = resp.DBInstance.DBSubnetGroup.DBSubnetGroupDescription
		}
		if resp.DBInstance.DBSubnetGroup.DBSubnetGroupName != nil {
			f27.DBSubnetGroupName = resp.DBInstance.DBSubnetGroup.DBSubnetGroupName
		}
		if resp.DBInstance.DBSubnetGroup.SubnetGroupStatus != nil {
			f27.SubnetGroupStatus = resp.DBInstance.DBSubnetGroup.SubnetGroupStatus
		}
		if resp.DBInstance.DBSubnetGroup.Subnets != nil {
			f27f4 := []*svcapitypes.Subnet{}
			for _, f27f4iter := range resp.DBInstance.DBSubnetGroup.Subnets {
				f27f4elem := &svcapitypes.Subnet{}
				if f27f4iter.SubnetAvailabilityZone != nil {
					f27f4elemf0 := &svcapitypes.AvailabilityZone{}
					if f27f4iter.SubnetAvailabilityZone.Name != nil {
						f27f4elemf0.Name = f27f4iter.SubnetAvailabilityZone.Name
					}
					f27f4elem.SubnetAvailabilityZone = f27f4elemf0
				}
				if f27f4iter.SubnetIdentifier != nil {
					f27f4elem.SubnetIdentifier = f27f4iter.SubnetIdentifier
				}
				if f27f4iter.SubnetOutpost != nil {
					f27f4elemf2 := &svcapitypes.Outpost{}
					if f27f4iter.SubnetOutpost.Arn != nil {
						f27f4elemf2.ARN = f27f4iter.SubnetOutpost.Arn
					}
					f27f4elem.SubnetOutpost = f27f4elemf2
				}
				if f27f4iter.SubnetStatus != nil {
					f27f4elem.SubnetStatus = f27f4iter.SubnetStatus
				}
				f27f4 = append(f27f4, f27f4elem)
			}
			f27.Subnets = f27f4
		}
		if resp.DBInstance.DBSubnetGroup.VpcId != nil {
			f27.VPCID = resp.DBInstance.DBSubnetGroup.VpcId
		}
		ko.Status.DBSubnetGroup = f27
	} else {
		ko.Status.DBSubnetGroup = nil
	}
	if resp.DBInstance.DbInstancePort != nil {
		ko.Status.DBInstancePort = resp.DBInstance.DbInstancePort
	} else {
		ko.Status.DBInstancePort = nil
	}
	if resp.DBInstance.DbiResourceId != nil {
		ko.Status.DBIResourceID = resp.DBInstance.DbiResourceId
	} else {
		ko.Status.DBIResourceID = nil
	}
	if resp.DBInstance.DeletionProtection != nil {
		ko.Spec.DeletionProtection = resp.DBInstance.DeletionProtection
	} else {
		ko.Spec.DeletionProtection = nil
	}
	if resp.DBInstance.DomainMemberships != nil {
		f31 := []*svcapitypes.DomainMembership{}
		for _, f31iter := range resp.DBInstance.DomainMemberships {
			f31elem := &svcapitypes.DomainMembership{}
			if f31iter.Domain != nil {
				f31elem.Domain = f31iter.Domain
			}
			if f31iter.FQDN != nil {
				f31elem.FQDN = f31iter.FQDN
			}
			if f31iter.IAMRoleName != nil {
				f31elem.IAMRoleName = f31iter.IAMRoleName
			}
			if f31iter.Status != nil {
				f31elem.Status = f31iter.Status
			}
			f31 = append(f31, f31elem)
		}
		ko.Status.DomainMemberships = f31
	} else {
		ko.Status.DomainMemberships = nil
	}
	if resp.DBInstance.EnabledCloudwatchLogsExports != nil {
		f32 := []*string{}
		for _, f32iter := range resp.DBInstance.EnabledCloudwatchLogsExports {
			var f32elem string
			f32elem = *f32iter
			f32 = append(f32, &f32elem)
		}
		ko.Status.EnabledCloudwatchLogsExports = f32
	} else {
		ko.Status.EnabledCloudwatchLogsExports = nil
	}
	if resp.DBInstance.Endpoint != nil {
		f33 := &svcapitypes.Endpoint{}
		if resp.DBInstance.Endpoint.Address != nil {
			f33.Address = resp.DBInstance.Endpoint.Address
		}
		if resp.DBInstance.Endpoint.HostedZoneId != nil {
			f33.HostedZoneID = resp.DBInstance.Endpoint.HostedZoneId
		}
		if resp.DBInstance.Endpoint.Port != nil {
			f33.Port = resp.DBInstance.Endpoint.Port
		}
		ko.Status.Endpoint = f33
	} else {
		ko.Status.Endpoint = nil
	}
	if resp.DBInstance.Engine != nil {
		ko.Spec.Engine = resp.DBInstance.Engine
	} else {
		ko.Spec.Engine = nil
	}
	if resp.DBInstance.EngineVersion != nil {
		ko.Spec.EngineVersion = resp.DBInstance.EngineVersion
	} else {
		ko.Spec.EngineVersion = nil
	}
	if resp.DBInstance.EnhancedMonitoringResourceArn != nil {
		ko.Status.EnhancedMonitoringResourceARN = resp.DBInstance.EnhancedMonitoringResourceArn
	} else {
		ko.Status.EnhancedMonitoringResourceARN = nil
	}
	if resp.DBInstance.IAMDatabaseAuthenticationEnabled != nil {
		ko.Status.IAMDatabaseAuthenticationEnabled = resp.DBInstance.IAMDatabaseAuthenticationEnabled
	} else {
		ko.Status.IAMDatabaseAuthenticationEnabled = nil
	}
	if resp.DBInstance.InstanceCreateTime != nil {
		ko.Status.InstanceCreateTime = &metav1.Time{*resp.DBInstance.InstanceCreateTime}
	} else {
		ko.Status.InstanceCreateTime = nil
	}
	if resp.DBInstance.Iops != nil {
		ko.Spec.IOPS = resp.DBInstance.Iops
	} else {
		ko.Spec.IOPS = nil
	}
	if resp.DBInstance.KmsKeyId != nil {
		ko.Spec.KMSKeyID = resp.DBInstance.KmsKeyId
	} else {
		ko.Spec.KMSKeyID = nil
	}
	if resp.DBInstance.LatestRestorableTime != nil {
		ko.Status.LatestRestorableTime = &metav1.Time{*resp.DBInstance.LatestRestorableTime}
	} else {
		ko.Status.LatestRestorableTime = nil
	}
	if resp.DBInstance.LicenseModel != nil {
		ko.Spec.LicenseModel = resp.DBInstance.LicenseModel
	} else {
		ko.Spec.LicenseModel = nil
	}
	if resp.DBInstance.ListenerEndpoint != nil {
		f43 := &svcapitypes.Endpoint{}
		if resp.DBInstance.ListenerEndpoint.Address != nil {
			f43.Address = resp.DBInstance.ListenerEndpoint.Address
		}
		if resp.DBInstance.ListenerEndpoint.HostedZoneId != nil {
			f43.HostedZoneID = resp.DBInstance.ListenerEndpoint.HostedZoneId
		}
		if resp.DBInstance.ListenerEndpoint.Port != nil {
			f43.Port = resp.DBInstance.ListenerEndpoint.Port
		}
		ko.Status.ListenerEndpoint = f43
	} else {
		ko.Status.ListenerEndpoint = nil
	}
	if resp.DBInstance.MasterUsername != nil {
		ko.Spec.MasterUsername = resp.DBInstance.MasterUsername
	} else {
		ko.Spec.MasterUsername = nil
	}
	if resp.DBInstance.MaxAllocatedStorage != nil {
		ko.Spec.MaxAllocatedStorage = resp.DBInstance.MaxAllocatedStorage
	} else {
		ko.Spec.MaxAllocatedStorage = nil
	}
	if resp.DBInstance.MonitoringInterval != nil {
		ko.Spec.MonitoringInterval = resp.DBInstance.MonitoringInterval
	} else {
		ko.Spec.MonitoringInterval = nil
	}
	if resp.DBInstance.MonitoringRoleArn != nil {
		ko.Spec.MonitoringRoleARN = resp.DBInstance.MonitoringRoleArn
	} else {
		ko.Spec.MonitoringRoleARN = nil
	}
	if resp.DBInstance.MultiAZ != nil {
		ko.Spec.MultiAZ = resp.DBInstance.MultiAZ
	} else {
		ko.Spec.MultiAZ = nil
	}
	if resp.DBInstance.NcharCharacterSetName != nil {
		ko.Spec.NcharCharacterSetName = resp.DBInstance.NcharCharacterSetName
	} else {
		ko.Spec.NcharCharacterSetName = nil
	}
	if resp.DBInstance.OptionGroupMemberships != nil {
		f50 := []*svcapitypes.OptionGroupMembership{}
		for _, f50iter := range resp.DBInstance.OptionGroupMemberships {
			f50elem := &svcapitypes.OptionGroupMembership{}
			if f50iter.OptionGroupName != nil {
				f50elem.OptionGroupName = f50iter.OptionGroupName
			}
			if f50iter.Status != nil {
				f50elem.Status = f50iter.Status
			}
			f50 = append(f50, f50elem)
		}
		ko.Status.OptionGroupMemberships = f50
	} else {
		ko.Status.OptionGroupMemberships = nil
	}
	if resp.DBInstance.PendingModifiedValues != nil {
		f51 := &svcapitypes.PendingModifiedValues{}
		if resp.DBInstance.PendingModifiedValues.AllocatedStorage != nil {
			f51.AllocatedStorage = resp.DBInstance.PendingModifiedValues.AllocatedStorage
		}
		if resp.DBInstance.PendingModifiedValues.AutomationMode != nil {
			f51.AutomationMode = resp.DBInstance.PendingModifiedValues.AutomationMode
		}
		if resp.DBInstance.PendingModifiedValues.BackupRetentionPeriod != nil {
			f51.BackupRetentionPeriod = resp.DBInstance.PendingModifiedValues.BackupRetentionPeriod
		}
		if resp.DBInstance.PendingModifiedValues.CACertificateIdentifier != nil {
			f51.CACertificateIdentifier = resp.DBInstance.PendingModifiedValues.CACertificateIdentifier
		}
		if resp.DBInstance.PendingModifiedValues.DBInstanceClass != nil {
			f51.DBInstanceClass = resp.DBInstance.PendingModifiedValues.DBInstanceClass
		}
		if resp.DBInstance.PendingModifiedValues.DBInstanceIdentifier != nil {
			f51.DBInstanceIdentifier = resp.DBInstance.PendingModifiedValues.DBInstanceIdentifier
		}
		if resp.DBInstance.PendingModifiedValues.DBSubnetGroupName != nil {
			f51.DBSubnetGroupName = resp.DBInstance.PendingModifiedValues.DBSubnetGroupName
		}
		if resp.DBInstance.PendingModifiedValues.EngineVersion != nil {
			f51.EngineVersion = resp.DBInstance.PendingModifiedValues.EngineVersion
		}
		if resp.DBInstance.PendingModifiedValues.IAMDatabaseAuthenticationEnabled != nil {
			f51.IAMDatabaseAuthenticationEnabled = resp.DBInstance.PendingModifiedValues.IAMDatabaseAuthenticationEnabled
		}
		if resp.DBInstance.PendingModifiedValues.Iops != nil {
			f51.IOPS = resp.DBInstance.PendingModifiedValues.Iops
		}
		if resp.DBInstance.PendingModifiedValues.LicenseModel != nil {
			f51.LicenseModel = resp.DBInstance.PendingModifiedValues.LicenseModel
		}
		if resp.DBInstance.PendingModifiedValues.MasterUserPassword != nil {
			f51.MasterUserPassword = resp.DBInstance.PendingModifiedValues.MasterUserPassword
		}
		if resp.DBInstance.PendingModifiedValues.MultiAZ != nil {
			f51.MultiAZ = resp.DBInstance.PendingModifiedValues.MultiAZ
		}
		if resp.DBInstance.PendingModifiedValues.PendingCloudwatchLogsExports != nil {
			f51f13 := &svcapitypes.PendingCloudwatchLogsExports{}
			if resp.DBInstance.PendingModifiedValues.PendingCloudwatchLogsExports.LogTypesToDisable != nil {
				f51f13f0 := []*string{}
				for _, f51f13f0iter := range resp.DBInstance.PendingModifiedValues.PendingCloudwatchLogsExports.LogTypesToDisable {
					var f51f13f0elem string
					f51f13f0elem = *f51f13f0iter
					f51f13f0 = append(f51f13f0, &f51f13f0elem)
				}
				f51f13.LogTypesToDisable = f51f13f0
			}
			if resp.DBInstance.PendingModifiedValues.PendingCloudwatchLogsExports.LogTypesToEnable != nil {
				f51f13f1 := []*string{}
				for _, f51f13f1iter := range resp.DBInstance.PendingModifiedValues.PendingCloudwatchLogsExports.LogTypesToEnable {
					var f51f13f1elem string
					f51f13f1elem = *f51f13f1iter
					f51f13f1 = append(f51f13f1, &f51f13f1elem)
				}
				f51f13.LogTypesToEnable = f51f13f1
			}
			f51.PendingCloudwatchLogsExports = f51f13
		}
		if resp.DBInstance.PendingModifiedValues.Port != nil {
			f51.Port = resp.DBInstance.PendingModifiedValues.Port
		}
		if resp.DBInstance.PendingModifiedValues.ProcessorFeatures != nil {
			f51f15 := []*svcapitypes.ProcessorFeature{}
			for _, f51f15iter := range resp.DBInstance.PendingModifiedValues.ProcessorFeatures {
				f51f15elem := &svcapitypes.ProcessorFeature{}
				if f51f15iter.Name != nil {
					f51f15elem.Name = f51f15iter.Name
				}
				if f51f15iter.Value != nil {
					f51f15elem.Value = f51f15iter.Value
				}
				f51f15 = append(f51f15, f51f15elem)
			}
			f51.ProcessorFeatures = f51f15
		}
		if resp.DBInstance.PendingModifiedValues.ResumeFullAutomationModeTime != nil {
			f51.ResumeFullAutomationModeTime = &metav1.Time{*resp.DBInstance.PendingModifiedValues.ResumeFullAutomationModeTime}
		}
		if resp.DBInstance.PendingModifiedValues.StorageType != nil {
			f51.StorageType = resp.DBInstance.PendingModifiedValues.StorageType
		}
		ko.Status.PendingModifiedValues = f51
	} else {
		ko.Status.PendingModifiedValues = nil
	}
	if resp.DBInstance.PerformanceInsightsEnabled != nil {
		ko.Spec.PerformanceInsightsEnabled = resp.DBInstance.PerformanceInsightsEnabled
	} else {
		ko.Spec.PerformanceInsightsEnabled = nil
	}
	if resp.DBInstance.PerformanceInsightsKMSKeyId != nil {
		ko.Spec.PerformanceInsightsKMSKeyID = resp.DBInstance.PerformanceInsightsKMSKeyId
	} else {
		ko.Spec.PerformanceInsightsKMSKeyID = nil
	}
	if resp.DBInstance.PerformanceInsightsRetentionPeriod != nil {
		ko.Spec.PerformanceInsightsRetentionPeriod = resp.DBInstance.PerformanceInsightsRetentionPeriod
	} else {
		ko.Spec.PerformanceInsightsRetentionPeriod = nil
	}
	if resp.DBInstance.PreferredBackupWindow != nil {
		ko.Spec.PreferredBackupWindow = resp.DBInstance.PreferredBackupWindow
	} else {
		ko.Spec.PreferredBackupWindow = nil
	}
	if resp.DBInstance.PreferredMaintenanceWindow != nil {
		ko.Spec.PreferredMaintenanceWindow = resp.DBInstance.PreferredMaintenanceWindow
	} else {
		ko.Spec.PreferredMaintenanceWindow = nil
	}
	if resp.DBInstance.ProcessorFeatures != nil {
		f57 := []*svcapitypes.ProcessorFeature{}
		for _, f57iter := range resp.DBInstance.ProcessorFeatures {
			f57elem := &svcapitypes.ProcessorFeature{}
			if f57iter.Name != nil {
				f57elem.Name = f57iter.Name
			}
			if f57iter.Value != nil {
				f57elem.Value = f57iter.Value
			}
			f57 = append(f57, f57elem)
		}
		ko.Spec.ProcessorFeatures = f57
	} else {
		ko.Spec.ProcessorFeatures = nil
	}
	if resp.DBInstance.PromotionTier != nil {
		ko.Spec.PromotionTier = resp.DBInstance.PromotionTier
	} else {
		ko.Spec.PromotionTier = nil
	}
	if resp.DBInstance.PubliclyAccessible != nil {
		ko.Spec.PubliclyAccessible = resp.DBInstance.PubliclyAccessible
	} else {
		ko.Spec.PubliclyAccessible = nil
	}
	if resp.DBInstance.ReadReplicaDBClusterIdentifiers != nil {
		f60 := []*string{}
		for _, f60iter := range resp.DBInstance.ReadReplicaDBClusterIdentifiers {
			var f60elem string
			f60elem = *f60iter
			f60 = append(f60, &f60elem)
		}
		ko.Status.ReadReplicaDBClusterIdentifiers = f60
	} else {
		ko.Status.ReadReplicaDBClusterIdentifiers = nil
	}
	if resp.DBInstance.ReadReplicaDBInstanceIdentifiers != nil {
		f61 := []*string{}
		for _, f61iter := range resp.DBInstance.ReadReplicaDBInstanceIdentifiers {
			var f61elem string
			f61elem = *f61iter
			f61 = append(f61, &f61elem)
		}
		ko.Status.ReadReplicaDBInstanceIdentifiers = f61
	} else {
		ko.Status.ReadReplicaDBInstanceIdentifiers = nil
	}
	if resp.DBInstance.ReadReplicaSourceDBInstanceIdentifier != nil {
		ko.Status.ReadReplicaSourceDBInstanceIdentifier = resp.DBInstance.ReadReplicaSourceDBInstanceIdentifier
	} else {
		ko.Status.ReadReplicaSourceDBInstanceIdentifier = nil
	}
	if resp.DBInstance.ReplicaMode != nil {
		ko.Status.ReplicaMode = resp.DBInstance.ReplicaMode
	} else {
		ko.Status.ReplicaMode = nil
	}
	if resp.DBInstance.ResumeFullAutomationModeTime != nil {
		ko.Status.ResumeFullAutomationModeTime = &metav1.Time{*resp.DBInstance.ResumeFullAutomationModeTime}
	} else {
		ko.Status.ResumeFullAutomationModeTime = nil
	}
	if resp.DBInstance.SecondaryAvailabilityZone != nil {
		ko.Status.SecondaryAvailabilityZone = resp.DBInstance.SecondaryAvailabilityZone
	} else {
		ko.Status.SecondaryAvailabilityZone = nil
	}
	if resp.DBInstance.StatusInfos != nil {
		f66 := []*svcapitypes.DBInstanceStatusInfo{}
		for _, f66iter := range resp.DBInstance.StatusInfos {
			f66elem := &svcapitypes.DBInstanceStatusInfo{}
			if f66iter.Message != nil {
				f66elem.Message = f66iter.Message
			}
			if f66iter.Normal != nil {
				f66elem.Normal = f66iter.Normal
			}
			if f66iter.Status != nil {
				f66elem.Status = f66iter.Status
			}
			if f66iter.StatusType != nil {
				f66elem.StatusType = f66iter.StatusType
			}
			f66 = append(f66, f66elem)
		}
		ko.Status.StatusInfos = f66
	} else {
		ko.Status.StatusInfos = nil
	}
	if resp.DBInstance.StorageEncrypted != nil {
		ko.Spec.StorageEncrypted = resp.DBInstance.StorageEncrypted
	} else {
		ko.Spec.StorageEncrypted = nil
	}
	if resp.DBInstance.StorageType != nil {
		ko.Spec.StorageType = resp.DBInstance.StorageType
	} else {
		ko.Spec.StorageType = nil
	}
	if resp.DBInstance.TagList != nil {
		f69 := []*svcapitypes.Tag{}
		for _, f69iter := range resp.DBInstance.TagList {
			f69elem := &svcapitypes.Tag{}
			if f69iter.Key != nil {
				f69elem.Key = f69iter.Key
			}
			if f69iter.Value != nil {
				f69elem.Value = f69iter.Value
			}
			f69 = append(f69, f69elem)
		}
		ko.Status.TagList = f69
	} else {
		ko.Status.TagList = nil
	}
	if resp.DBInstance.TdeCredentialArn != nil {
		ko.Spec.TDECredentialARN = resp.DBInstance.TdeCredentialArn
	} else {
		ko.Spec.TDECredentialARN = nil
	}
	if resp.DBInstance.Timezone != nil {
		ko.Spec.Timezone = resp.DBInstance.Timezone
	} else {
		ko.Spec.Timezone = nil
	}
	if resp.DBInstance.VpcSecurityGroups != nil {
		f72 := []*svcapitypes.VPCSecurityGroupMembership{}
		for _, f72iter := range resp.DBInstance.VpcSecurityGroups {
			f72elem := &svcapitypes.VPCSecurityGroupMembership{}
			if f72iter.Status != nil {
				f72elem.Status = f72iter.Status
			}
			if f72iter.VpcSecurityGroupId != nil {
				f72elem.VPCSecurityGroupID = f72iter.VpcSecurityGroupId
			}
			f72 = append(f72, f72elem)
		}
		ko.Status.VPCSecurityGroups = f72
	} else {
		ko.Status.VPCSecurityGroups = nil
	}

	rm.setStatusDefaults(ko)

	// We expect the DB instance to be in 'creating' status since we just
	// issued the call to create it, but I suppose it doesn't hurt to check
	// here.
	if instanceCreating(&resource{ko}) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
		return &resource{ko}, nil
	}
	return &resource{ko}, nil
}

// restoreDbInstanceFromDbSnapshotPayload returns an SDK-specific struct for the HTTP request
// payload of the RestoreDbInstanceFromDbSnapshot API call
func (rm *resourceManager) restoreDbInstanceFromDbSnapshotPayload(
	r *resource,
) (*svcsdk.CreateDBInstanceInput, error) {
	res := &svcsdk.CreateDBInstanceInput{}

	if r.ko.Spec.DBSnapshotIdentifier != nil {
		res.SetDBSnapshotIdentifier(*r.ko.Spec.DBSnapshotIdentifier)
	}

	return res, nil
}
