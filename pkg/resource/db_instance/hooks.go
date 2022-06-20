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

	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	corev1 "k8s.io/api/core/v1"
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

// function to create restoreDbInstanceFromDbSnapshot payload and call restoreDbInstanceFromDbSnapshot API
func (rm *resourceManager) restoreDbInstanceFromDbSnapshot(
	ctx context.Context,
	r *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.restoreDbInstanceFromDbSnapshot")
	defer func(err error) { exit(err) }(err)

	resp, respErr := rm.sdkapi.RestoreDBInstanceFromDBSnapshotWithContext(ctx, rm.newRestoreDBInstanceFromDBSnapshotInput(r))
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
		return &resource{r.ko}, nil
	}
	return &resource{r.ko}, nil
}

// function to create createDBInstanceReadReplica payload and call createDBInstanceReadReplica API
func (rm *resourceManager) createDBInstanceReadReplica(
	ctx context.Context,
	r *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.createDBInstanceReadReplica")
	defer func(err error) { exit(err) }(err)

	resp, respErr := rm.sdkapi.CreateDBInstanceReadReplicaWithContext(ctx, rm.newCreateDBInstanceReadReplicaInput(r))
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
		return &resource{r.ko}, nil
	}
	return &resource{r.ko}, nil

// RDS will choose preferred engine minor version if only
// engine major version is provided and controler should not
// treat them as different, such as spec has 14, status has 14.1
// controller should treat them as same
func reconcileEngineVersion(
	a *resource,
	b *resource,
) {
	if a != nil && b != nil && strings.HasPrefix(*b.ko.Spec.EngineVersion, *a.ko.Spec.EngineVersion) {
		a.ko.Spec.EngineVersion = b.ko.Spec.EngineVersion
	}
}
