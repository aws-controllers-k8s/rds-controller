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

package db_subnet_group

import (
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackerrors "github.com/aws-controllers-k8s/runtime/pkg/errors"
	acktypes "github.com/aws-controllers-k8s/runtime/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8srt "k8s.io/apimachinery/pkg/runtime"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

// Hack to avoid import errors during build...
var (
	_ = &ackerrors.MissingNameIdentifier
)

// resource implements the `aws-controller-k8s/runtime/pkg/types.AWSResource`
// interface
type resource struct {
	// The Kubernetes-native CR representing the resource
	ko *svcapitypes.DBSubnetGroup
}

// Identifiers returns an AWSResourceIdentifiers object containing various
// identifying information, including the AWS account ID that owns the
// resource, the resource's AWS Resource Name (ARN)
func (r *resource) Identifiers() acktypes.AWSResourceIdentifiers {
	return &resourceIdentifiers{r.ko.Status.ACKResourceMetadata}
}

// IsBeingDeleted returns true if the Kubernetes resource has a non-zero
// deletion timestemp
func (r *resource) IsBeingDeleted() bool {
	return !r.ko.DeletionTimestamp.IsZero()
}

// RuntimeObject returns the Kubernetes apimachinery/runtime representation of
// the AWSResource
func (r *resource) RuntimeObject() k8srt.Object {
	return r.ko
}

// MetaObject returns the Kubernetes apimachinery/apis/meta/v1.Object
// representation of the AWSResource
func (r *resource) MetaObject() metav1.Object {
	return r.ko
}

// RuntimeMetaObject returns an object that implements both the Kubernetes
// apimachinery/runtime.Object and the Kubernetes
// apimachinery/apis/meta/v1.Object interfaces
func (r *resource) RuntimeMetaObject() acktypes.RuntimeMetaObject {
	return r.ko
}

// Conditions returns the ACK Conditions collection for the AWSResource
func (r *resource) Conditions() []*ackv1alpha1.Condition {
	return r.ko.Status.Conditions
}

// ReplaceConditions sets the Conditions status field for the resource
func (r *resource) ReplaceConditions(conditions []*ackv1alpha1.Condition) {
	r.ko.Status.Conditions = conditions
}

// SetObjectMeta sets the ObjectMeta field for the resource
func (r *resource) SetObjectMeta(meta metav1.ObjectMeta) {
	r.ko.ObjectMeta = meta
}

// SetIdentifiers sets the Spec or Status field that is referenced as the unique
// resource identifier
func (r *resource) SetIdentifiers(identifier *ackv1alpha1.AWSIdentifiers) error {
	if identifier.NameOrID == "" {
		return ackerrors.MissingNameIdentifier
	}
	r.ko.Spec.Name = &identifier.NameOrID

	return nil
}
