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

package db_cluster_endpoint

import (
	"bytes"
	"reflect"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	acktags "github.com/aws-controllers-k8s/runtime/pkg/tags"
)

// Hack to avoid import errors during build...
var (
	_ = &bytes.Buffer{}
	_ = &reflect.Method{}
	_ = &acktags.Tags{}
)

// newResourceDelta returns a new `ackcompare.Delta` used to compare two
// resources
func newResourceDelta(
	a *resource,
	b *resource,
) *ackcompare.Delta {
	delta := ackcompare.NewDelta()
	if (a == nil && b != nil) ||
		(a != nil && b == nil) {
		delta.Add("", a, b)
		return delta
	}
	compareTags(delta, a, b)

	if ackcompare.HasNilDifference(a.ko.Spec.DBClusterEndpointIdentifier, b.ko.Spec.DBClusterEndpointIdentifier) {
		delta.Add("Spec.DBClusterEndpointIdentifier", a.ko.Spec.DBClusterEndpointIdentifier, b.ko.Spec.DBClusterEndpointIdentifier)
	} else if a.ko.Spec.DBClusterEndpointIdentifier != nil && b.ko.Spec.DBClusterEndpointIdentifier != nil {
		if *a.ko.Spec.DBClusterEndpointIdentifier != *b.ko.Spec.DBClusterEndpointIdentifier {
			delta.Add("Spec.DBClusterEndpointIdentifier", a.ko.Spec.DBClusterEndpointIdentifier, b.ko.Spec.DBClusterEndpointIdentifier)
		}
	}
	if ackcompare.HasNilDifference(a.ko.Spec.DBClusterIdentifier, b.ko.Spec.DBClusterIdentifier) {
		delta.Add("Spec.DBClusterIdentifier", a.ko.Spec.DBClusterIdentifier, b.ko.Spec.DBClusterIdentifier)
	} else if a.ko.Spec.DBClusterIdentifier != nil && b.ko.Spec.DBClusterIdentifier != nil {
		if *a.ko.Spec.DBClusterIdentifier != *b.ko.Spec.DBClusterIdentifier {
			delta.Add("Spec.DBClusterIdentifier", a.ko.Spec.DBClusterIdentifier, b.ko.Spec.DBClusterIdentifier)
		}
	}
	if !reflect.DeepEqual(a.ko.Spec.DBClusterIdentifierRef, b.ko.Spec.DBClusterIdentifierRef) {
		delta.Add("Spec.DBClusterIdentifierRef", a.ko.Spec.DBClusterIdentifierRef, b.ko.Spec.DBClusterIdentifierRef)
	}
	if ackcompare.HasNilDifference(a.ko.Spec.EndpointType, b.ko.Spec.EndpointType) {
		delta.Add("Spec.EndpointType", a.ko.Spec.EndpointType, b.ko.Spec.EndpointType)
	} else if a.ko.Spec.EndpointType != nil && b.ko.Spec.EndpointType != nil {
		if *a.ko.Spec.EndpointType != *b.ko.Spec.EndpointType {
			delta.Add("Spec.EndpointType", a.ko.Spec.EndpointType, b.ko.Spec.EndpointType)
		}
	}
	if len(a.ko.Spec.ExcludedMembers) != len(b.ko.Spec.ExcludedMembers) {
		delta.Add("Spec.ExcludedMembers", a.ko.Spec.ExcludedMembers, b.ko.Spec.ExcludedMembers)
	} else if len(a.ko.Spec.ExcludedMembers) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.ExcludedMembers, b.ko.Spec.ExcludedMembers) {
			delta.Add("Spec.ExcludedMembers", a.ko.Spec.ExcludedMembers, b.ko.Spec.ExcludedMembers)
		}
	}
	if len(a.ko.Spec.StaticMembers) != len(b.ko.Spec.StaticMembers) {
		delta.Add("Spec.StaticMembers", a.ko.Spec.StaticMembers, b.ko.Spec.StaticMembers)
	} else if len(a.ko.Spec.StaticMembers) > 0 {
		if !ackcompare.SliceStringPEqual(a.ko.Spec.StaticMembers, b.ko.Spec.StaticMembers) {
			delta.Add("Spec.StaticMembers", a.ko.Spec.StaticMembers, b.ko.Spec.StaticMembers)
		}
	}

	return delta
}
