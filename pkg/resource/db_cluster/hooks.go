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
	"errors"
	"fmt"

	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
)

// NOTE(jaypipes): The below list is derived from looking at the RDS control
// plane source code. I've spoken with the RDS team about not having a list of
// publicly-visible DBClusterStatus strings available through the API model
// and they are looking into doing that. If we get that, we could use the
// constants generated in the `apis/{VERSION}/enums.go` file.
const (
	StatusAvailable                         = "available"
	StatusBackingUp                         = "backing-up"
	StatusCreating                          = "creating"
	StatusDeleting                          = "deleting"
	StatusFailed                            = "failed"
	StatusFailingOver                       = "failing-over"
	StatusInaccessibleEncryptionCredentials = "inaccessible-encryption-credentials"
	StatusBacktracking                      = "backtracking"
	StatusMaintenance                       = "maintenance"
	StatusMigrating                         = "migrating"
	StatusModifying                         = "modifying"
	StatusMigrationFailed                   = "migration-failed"
	StatusRestoreFailed                     = "restore-failed"
	StatusPreparingDataMigration            = "preparing-data-migration"
	StatusPromoting                         = "promoting"
	StatusRebooting                         = "rebooting"
	StatusResettingMasterCredentials        = "resetting-master-credentials"
	StatusRenaming                          = "renaming"
	StatusScalingStorage                    = "scaling-storage"
	StatusScalingCompute                    = "scaling-compute"
	StatusScalingCapacity                   = "scaling-capacity"
	StatusUpgrading                         = "upgrading"
	StatusConfiguringIAMDatabaseAuthr       = "configuring-iam-database-auth"
	StatusIncompatibleParameters            = "incompatible-parameters"
	StatusStarting                          = "starting"
	StatusStopping                          = "stopping"
	StatusStopped                           = "stopped"
	StatusStorageFailure                    = "storage-failure"
	StatusIncompatibleRestore               = "incompatible-restore"
	StatusIncompatibleNetwork               = "incompatible-network"
	StatusInsufficientResourceLimits        = "insufficient-resource-limits"
	StatusInsufficientCapacity              = "insufficient-capacity"
	StatusReplicationSuspended              = "replication-suspended"
	StatusConfiguringActivityStream         = "configuring-activity-stream"
	StatusArchiving                         = "archiving"
	StatusArchived                          = "archived"
)

var (
	// TerminalStatuses are the status strings that are terminal states for a
	// DB cluster.
	TerminalStatuses = []string{
		StatusDeleting,
		StatusInaccessibleEncryptionCredentials,
		StatusIncompatibleNetwork,
		StatusIncompatibleRestore,
		StatusFailed,
	}
)

var (
	requeueWaitWhileDeleting = ackrequeue.NeededAfter(
		errors.New("DB cluster in 'deleting' state, cannot be modified or deleted."),
		ackrequeue.DefaultRequeueAfterDuration,
	)
)

// requeueWaitUntilCanModify returns a `ackrequeue.RequeueNeededAfter` struct
// explaining the DB instance cannot be modified until it reaches an available
// status.
func requeueWaitUntilCanModify(r *resource) *ackrequeue.RequeueNeededAfter {
	if r.ko.Status.Status == nil {
		return nil
	}
	status := *r.ko.Status.Status
	msg := fmt.Sprintf(
		"DB cluster in '%s' state, cannot be modified until '%s'.",
		status, StatusAvailable,
	)
	return ackrequeue.NeededAfter(
		errors.New(msg),
		ackrequeue.DefaultRequeueAfterDuration,
	)
}

// clusterHasTerminalStatus returns whether the supplied DB Cluster is in a
// terminal state
func clusterHasTerminalStatus(r *resource) bool {
	if r.ko.Status.Status == nil {
		return false
	}
	dbcs := *r.ko.Status.Status
	for _, s := range TerminalStatuses {
		if dbcs == s {
			return true
		}
	}
	return false
}

// clusterCreating returns true if the supplied DB cluster is in the process
// of being created
func clusterCreating(r *resource) bool {
	if r.ko.Status.Status == nil {
		return false
	}
	dbcs := *r.ko.Status.Status
	return dbcs == StatusCreating
}

// clusterDeleting returns true if the supplied DB cluster is in the process
// of being deleted
func clusterDeleting(r *resource) bool {
	if r.ko.Status.Status == nil {
		return false
	}
	dbcs := *r.ko.Status.Status
	return dbcs == StatusDeleting
}
