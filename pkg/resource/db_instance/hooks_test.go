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
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

// TestSetIAMDatabaseAuthenticationFromStatus validates that a change of
// spec.enableIAMDatabaseAuthentication after creation is visible in the delta
// once the observed state has been copied from the status of the latest
// resource, which sdkFind builds from a copy of the desired resource.
func TestSetIAMDatabaseAuthenticationFromStatus(t *testing.T) {
	tests := []struct {
		name            string
		desired         svcapitypes.DBInstanceSpec
		observed        *bool
		expectDifferent bool
	}{
		{
			name: "enabling after creation is detected",
			desired: svcapitypes.DBInstanceSpec{
				EnableIAMDatabaseAuthentication: aws.Bool(true),
			},
			observed:        aws.Bool(false),
			expectDifferent: true,
		},
		{
			name: "disabling after creation is detected",
			desired: svcapitypes.DBInstanceSpec{
				EnableIAMDatabaseAuthentication: aws.Bool(false),
			},
			observed:        aws.Bool(true),
			expectDifferent: true,
		},
		{
			name: "matching state is not a difference",
			desired: svcapitypes.DBInstanceSpec{
				EnableIAMDatabaseAuthentication: aws.Bool(true),
			},
			observed:        aws.Bool(true),
			expectDifferent: false,
		},
		{
			name:            "unset in the spec is left alone",
			desired:         svcapitypes.DBInstanceSpec{},
			observed:        aws.Bool(true),
			expectDifferent: false,
		},
		{
			name: "cluster members are managed by the DB cluster",
			desired: svcapitypes.DBInstanceSpec{
				DBClusterIdentifier:             aws.String("my-cluster"),
				EnableIAMDatabaseAuthentication: aws.Bool(true),
			},
			observed:        aws.Bool(false),
			expectDifferent: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			desired := &resource{
				ko: &svcapitypes.DBInstance{
					Spec: tc.desired,
				},
			}
			latest := &resource{
				ko: desired.ko.DeepCopy(),
			}
			latest.ko.Status.IAMDatabaseAuthenticationEnabled = tc.observed

			setIAMDatabaseAuthenticationFromStatus(desired, latest)

			delta := newResourceDelta(desired, latest)
			assert.Equal(t, tc.expectDifferent, delta.DifferentAt("Spec.EnableIAMDatabaseAuthentication"),
				"unexpected Spec.EnableIAMDatabaseAuthentication delta result for case %q", tc.name)
		})
	}
}
