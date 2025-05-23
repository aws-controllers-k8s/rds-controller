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

package db_cluster_snapshot

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
	if ackcompare.HasNilDifference(a.ko.Spec.DBClusterSnapshotIdentifier, b.ko.Spec.DBClusterSnapshotIdentifier) {
		delta.Add("Spec.DBClusterSnapshotIdentifier", a.ko.Spec.DBClusterSnapshotIdentifier, b.ko.Spec.DBClusterSnapshotIdentifier)
	} else if a.ko.Spec.DBClusterSnapshotIdentifier != nil && b.ko.Spec.DBClusterSnapshotIdentifier != nil {
		if *a.ko.Spec.DBClusterSnapshotIdentifier != *b.ko.Spec.DBClusterSnapshotIdentifier {
			delta.Add("Spec.DBClusterSnapshotIdentifier", a.ko.Spec.DBClusterSnapshotIdentifier, b.ko.Spec.DBClusterSnapshotIdentifier)
		}
	}
	desiredACKTags, _ := convertToOrderedACKTags(a.ko.Spec.Tags)
	latestACKTags, _ := convertToOrderedACKTags(b.ko.Spec.Tags)
	if !ackcompare.MapStringStringEqual(desiredACKTags, latestACKTags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	}

	return delta
}
