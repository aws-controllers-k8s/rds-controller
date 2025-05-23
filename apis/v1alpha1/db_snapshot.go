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

// Code generated by ack-generate. DO NOT EDIT.

package v1alpha1

import (
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DBSnapshotSpec defines the desired state of DBSnapshot.
//
// Contains the details of an Amazon RDS DB snapshot.
//
// This data type is used as a response element in the DescribeDBSnapshots action.
type DBSnapshotSpec struct {

	// The identifier of the DB instance that you want to create the snapshot of.
	//
	// Constraints:
	//
	//   - Must match the identifier of an existing DBInstance.
	DBInstanceIdentifier    *string                                  `json:"dbInstanceIdentifier,omitempty"`
	DBInstanceIdentifierRef *ackv1alpha1.AWSResourceReferenceWrapper `json:"dbInstanceIdentifierRef,omitempty"`
	// The identifier for the DB snapshot.
	//
	// Constraints:
	//
	//   - Can't be null, empty, or blank
	//
	//   - Must contain from 1 to 255 letters, numbers, or hyphens
	//
	//   - First character must be a letter
	//
	//   - Can't end with a hyphen or contain two consecutive hyphens
	//
	// Example: my-snapshot-id
	// +kubebuilder:validation:Required
	DBSnapshotIdentifier *string `json:"dbSnapshotIdentifier"`
	Tags                 []*Tag  `json:"tags,omitempty"`
}

// DBSnapshotStatus defines the observed state of DBSnapshot
type DBSnapshotStatus struct {
	// All CRs managed by ACK have a common `Status.ACKResourceMetadata` member
	// that is used to contain resource sync state, account ownership,
	// constructed ARN for the resource
	// +kubebuilder:validation:Optional
	ACKResourceMetadata *ackv1alpha1.ResourceMetadata `json:"ackResourceMetadata"`
	// All CRs managed by ACK have a common `Status.Conditions` member that
	// contains a collection of `ackv1alpha1.Condition` objects that describe
	// the various terminal states of the CR and its backend AWS service API
	// resource
	// +kubebuilder:validation:Optional
	Conditions []*ackv1alpha1.Condition `json:"conditions"`
	// Specifies the allocated storage size in gibibytes (GiB).
	// +kubebuilder:validation:Optional
	AllocatedStorage *int64 `json:"allocatedStorage,omitempty"`
	// Specifies the name of the Availability Zone the DB instance was located in
	// at the time of the DB snapshot.
	// +kubebuilder:validation:Optional
	AvailabilityZone *string `json:"availabilityZone,omitempty"`
	// The identifier for the source DB instance, which can't be changed and which
	// is unique to an Amazon Web Services Region.
	// +kubebuilder:validation:Optional
	DBIResourceID *string `json:"dbiResourceID,omitempty"`
	// Indicates whether the DB snapshot is encrypted.
	// +kubebuilder:validation:Optional
	Encrypted *bool `json:"encrypted,omitempty"`
	// Specifies the name of the database engine.
	// +kubebuilder:validation:Optional
	Engine *string `json:"engine,omitempty"`
	// Indicates whether mapping of Amazon Web Services Identity and Access Management
	// (IAM) accounts to database accounts is enabled.
	// +kubebuilder:validation:Optional
	IAMDatabaseAuthenticationEnabled *bool `json:"iamDatabaseAuthenticationEnabled,omitempty"`
	// Specifies the time in Coordinated Universal Time (UTC) when the DB instance,
	// from which the snapshot was taken, was created.
	// +kubebuilder:validation:Optional
	InstanceCreateTime *metav1.Time `json:"instanceCreateTime,omitempty"`
	// Specifies the Provisioned IOPS (I/O operations per second) value of the DB
	// instance at the time of the snapshot.
	// +kubebuilder:validation:Optional
	IOPS *int64 `json:"iops,omitempty"`
	// If Encrypted is true, the Amazon Web Services KMS key identifier for the
	// encrypted DB snapshot.
	//
	// The Amazon Web Services KMS key identifier is the key ARN, key ID, alias
	// ARN, or alias name for the KMS key.
	// +kubebuilder:validation:Optional
	KMSKeyID *string `json:"kmsKeyID,omitempty"`
	// License model information for the restored DB instance.
	// +kubebuilder:validation:Optional
	LicenseModel *string `json:"licenseModel,omitempty"`
	// Provides the master username for the DB snapshot.
	// +kubebuilder:validation:Optional
	MasterUsername *string `json:"masterUsername,omitempty"`
	// Specifies the time of the CreateDBSnapshot operation in Coordinated Universal
	// Time (UTC). Doesn't change when the snapshot is copied.
	// +kubebuilder:validation:Optional
	OriginalSnapshotCreateTime *metav1.Time `json:"originalSnapshotCreateTime,omitempty"`
	// The percentage of the estimated data that has been transferred.
	// +kubebuilder:validation:Optional
	PercentProgress *int64 `json:"percentProgress,omitempty"`
	// Specifies the port that the database engine was listening on at the time
	// of the snapshot.
	// +kubebuilder:validation:Optional
	Port *int64 `json:"port,omitempty"`
	// The number of CPU cores and the number of threads per core for the DB instance
	// class of the DB instance when the DB snapshot was created.
	// +kubebuilder:validation:Optional
	ProcessorFeatures []*ProcessorFeature `json:"processorFeatures,omitempty"`
	// Specifies when the snapshot was taken in Coordinated Universal Time (UTC).
	// Changes for the copy when the snapshot is copied.
	// +kubebuilder:validation:Optional
	SnapshotCreateTime *metav1.Time `json:"snapshotCreateTime,omitempty"`
	// The timestamp of the most recent transaction applied to the database that
	// you're backing up. Thus, if you restore a snapshot, SnapshotDatabaseTime
	// is the most recent transaction in the restored DB instance. In contrast,
	// originalSnapshotCreateTime specifies the system time that the snapshot completed.
	//
	// If you back up a read replica, you can determine the replica lag by comparing
	// SnapshotDatabaseTime with originalSnapshotCreateTime. For example, if originalSnapshotCreateTime
	// is two hours later than SnapshotDatabaseTime, then the replica lag is two
	// hours.
	// +kubebuilder:validation:Optional
	SnapshotDatabaseTime *metav1.Time `json:"snapshotDatabaseTime,omitempty"`
	// Specifies where manual snapshots are stored: Amazon Web Services Outposts
	// or the Amazon Web Services Region.
	// +kubebuilder:validation:Optional
	SnapshotTarget *string `json:"snapshotTarget,omitempty"`
	// Provides the type of the DB snapshot.
	// +kubebuilder:validation:Optional
	SnapshotType *string `json:"snapshotType,omitempty"`
	// The DB snapshot Amazon Resource Name (ARN) that the DB snapshot was copied
	// from. It only has a value in the case of a cross-account or cross-Region
	// copy.
	// +kubebuilder:validation:Optional
	SourceDBSnapshotIdentifier *string `json:"sourceDBSnapshotIdentifier,omitempty"`
	// The Amazon Web Services Region that the DB snapshot was created in or copied
	// from.
	// +kubebuilder:validation:Optional
	SourceRegion *string `json:"sourceRegion,omitempty"`
	// Specifies the status of this DB snapshot.
	// +kubebuilder:validation:Optional
	Status *string `json:"status,omitempty"`
	// Specifies the storage throughput for the DB snapshot.
	// +kubebuilder:validation:Optional
	StorageThroughput *int64 `json:"storageThroughput,omitempty"`
	// Specifies the storage type associated with DB snapshot.
	// +kubebuilder:validation:Optional
	StorageType *string `json:"storageType,omitempty"`
	// +kubebuilder:validation:Optional
	TagList []*Tag `json:"tagList,omitempty"`
	// The ARN from the key store with which to associate the instance for TDE encryption.
	// +kubebuilder:validation:Optional
	TDECredentialARN *string `json:"tdeCredentialARN,omitempty"`
	// The time zone of the DB snapshot. In most cases, the Timezone element is
	// empty. Timezone content appears only for snapshots taken from Microsoft SQL
	// Server DB instances that were created with a time zone specified.
	// +kubebuilder:validation:Optional
	Timezone *string `json:"timezone,omitempty"`
	// Provides the VPC ID associated with the DB snapshot.
	// +kubebuilder:validation:Optional
	VPCID *string `json:"vpcID,omitempty"`
}

// DBSnapshot is the Schema for the DBSnapshots API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type DBSnapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DBSnapshotSpec   `json:"spec,omitempty"`
	Status            DBSnapshotStatus `json:"status,omitempty"`
}

// DBSnapshotList contains a list of DBSnapshot
// +kubebuilder:object:root=true
type DBSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DBSnapshot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DBSnapshot{}, &DBSnapshotList{})
}
