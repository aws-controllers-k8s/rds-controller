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

package util_test

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/rds-controller/pkg/util"
)

var (
	tagA  = &svcapitypes.Tag{Key: aws.String("A"), Value: aws.String("1")}
	tagB  = &svcapitypes.Tag{Key: aws.String("B"), Value: aws.String("2")}
	tagC  = &svcapitypes.Tag{Key: aws.String("C"), Value: aws.String("3")}
	tagD  = &svcapitypes.Tag{Key: aws.String("D"), Value: aws.String("4")}
	tagE  = &svcapitypes.Tag{Key: aws.String("E"), Value: aws.String("5")}
	tagE2 = &svcapitypes.Tag{Key: aws.String("E"), Value: aws.String("6")}
)

func TestComputeTagsDelta(t *testing.T) {
	type args struct {
		a []*svcapitypes.Tag
		b []*svcapitypes.Tag
	}
	tests := []struct {
		name               string
		args               args
		wantAddedOrUpdated []*svcapitypes.Tag
		wantRemoved        []string
	}{
		{
			name:               "empty arrays",
			args:               args{},
			wantAddedOrUpdated: nil,
			wantRemoved:        nil,
		},
		{
			name: "only added tags",
			args: args{
				a: []*svcapitypes.Tag{tagA, tagB, tagC},
			},
			wantAddedOrUpdated: []*svcapitypes.Tag{tagA, tagB, tagC},
			wantRemoved:        nil,
		},
		{
			name: "only removed tags",
			args: args{
				b: []*svcapitypes.Tag{tagA, tagB, tagC},
			},
			wantAddedOrUpdated: nil,
			wantRemoved:        []string{"A", "B", "C"},
		},
		{
			name: "added and removed tags",
			args: args{
				a: []*svcapitypes.Tag{tagD, tagE},
				b: []*svcapitypes.Tag{tagA, tagB, tagC},
			},
			wantAddedOrUpdated: []*svcapitypes.Tag{tagD, tagE},
			wantRemoved:        []string{"A", "B", "C"},
		},
		{
			name: "added, updated and removed tags",
			args: args{
				a: []*svcapitypes.Tag{tagD, tagE2},
				b: []*svcapitypes.Tag{tagA, tagB, tagC, tagE},
			},
			// notice the order of b is not the same.
			wantAddedOrUpdated: []*svcapitypes.Tag{tagE2, tagD},
			wantRemoved:        []string{"A", "B", "C"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAddedOrUpdated, gotRemoved := util.ComputeTagsDelta(tt.args.a, tt.args.b)
			if !reflect.DeepEqual(gotAddedOrUpdated, tt.wantAddedOrUpdated) {
				t.Errorf("ComputeTagsDelta() gotAddedOrUpdated = %v, want %v", gotAddedOrUpdated, tt.wantAddedOrUpdated)
			}
			if !reflect.DeepEqual(gotRemoved, tt.wantRemoved) {
				t.Errorf("ComputeTagsDelta() gotRemoved = %v, want %v", gotRemoved, tt.wantRemoved)
			}
		})
	}
}
