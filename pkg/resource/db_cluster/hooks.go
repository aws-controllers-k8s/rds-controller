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
	"errors"
	"fmt"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go/service/rds"
	corev1 "k8s.io/api/core/v1"

	"github.com/aws-controllers-k8s/rds-controller/pkg/util"
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

// clusterAvailable returns true if the supplied DB cluster is in an
// available status
func clusterAvailable(r *resource) bool {
	if r.ko.Status.Status == nil {
		return false
	}
	dbcs := *r.ko.Status.Status
	return dbcs == StatusAvailable
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
//     `ResourceName`, but actually expects an ARN, not the cluster
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
		rlog.Debug("removing tags from cluster", "tags", toDelete)
		_, err = rm.sdkapi.RemoveTagsFromResourceWithContext(
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
		rlog.Debug("adding tags to cluster", "tags", toAdd)
		_, err = rm.sdkapi.AddTagsToResourceWithContext(
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
	resp, err := rm.sdkapi.ListTagsForResourceWithContext(
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
) []*svcsdk.Tag {
	tags := make([]*svcsdk.Tag, len(rTags))
	for i := range rTags {
		tags[i] = &svcsdk.Tag{
			Key:   rTags[i].Key,
			Value: rTags[i].Value,
		}
	}
	return tags
}

// function to create restoreDbClusterFromSnapshot payload and call restoreDbClusterFromSnapshot API
func (rm *resourceManager) restoreDbClusterFromSnapshot(
	ctx context.Context,
	r *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.restoreDbClusterFromSnapshot")
	defer func(err error) { exit(err) }(err)

	resp, respErr := rm.sdkapi.RestoreDBClusterFromSnapshotWithContext(ctx, rm.newRestoreDBClusterFromSnapshotInput(r))
	rm.metrics.RecordAPICall("CREATE", "RestoreDbClusterFromSnapshot", respErr)
	if respErr != nil {
		return nil, respErr
	}

	rm.setResourceFromRestoreDBClusterFromSnapshotOutput(r, resp)
	rm.setStatusDefaults(r.ko)

	// We expect the DB cluster to be in 'creating' status since we just
	// issued the call to create it, but I suppose it doesn't hurt to check
	// here.
	if clusterCreating(&resource{r.ko}) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(&resource{r.ko}, corev1.ConditionFalse, nil, nil)
	}
	return &resource{r.ko}, nil
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
