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

package util

import (
	ackutil "github.com/aws-controllers-k8s/runtime/pkg/util"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

// TODO(a-hilaly) most of the utility in this package should ideally go to
// ack runtime or pkg repository.

// computeTagsDelta compares two Tag arrays and return two different list
// containing the addedOrupdated and removed tags. The removed tags array
// only contains the tags Keys.
func ComputeTagsDelta(
	a []*svcapitypes.Tag,
	b []*svcapitypes.Tag,
) (addedOrUpdated []*svcapitypes.Tag, removed []string) {
	var visitedIndexes []string
mainLoop:
	for _, aElement := range b {
		visitedIndexes = append(visitedIndexes, *aElement.Key)
		for _, bElement := range a {
			if equalStrings(aElement.Key, bElement.Key) {
				if !equalStrings(aElement.Value, bElement.Value) {
					addedOrUpdated = append(addedOrUpdated, bElement)
				}
				continue mainLoop
			}
		}
		removed = append(removed, *aElement.Key)
	}
	for _, bElement := range a {
		if !ackutil.InStrings(*bElement.Key, visitedIndexes) {
			addedOrUpdated = append(addedOrUpdated, bElement)
		}
	}
	return addedOrUpdated, removed
}

// equalTags returns true if two Tag arrays are equal regardless of the order
// of their elements.
func EqualTags(
	a []*svcapitypes.Tag,
	b []*svcapitypes.Tag,
) bool {
	addedOrUpdated, removed := ComputeTagsDelta(a, b)
	return len(addedOrUpdated) == 0 && len(removed) == 0
}

func equalStrings(a, b *string) bool {
	if a == nil {
		return b == nil || *b == ""
	}
	return (*a == "" && b == nil) || *a == *b
}
