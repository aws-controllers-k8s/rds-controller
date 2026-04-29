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
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

func TestNewResourceDelta_EngineVersion(t *testing.T) {
	tests := []struct {
		name                    string
		desiredEngineVersion    string
		latestEngineVersion     string
		autoMinorVersionUpgrade *bool
		expectDifferentAt       bool
	}{
		{
			name:                    "same full aurora version, auto minor upgrade enabled",
			desiredEngineVersion:    "8.0.mysql_aurora.3.10.3",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       false,
		},
		{
			name:                    "same full aurora version, auto minor upgrade disabled",
			desiredEngineVersion:    "8.0.mysql_aurora.3.10.3",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: aws.Bool(false),
			expectDifferentAt:       false,
		},
		{
			name:                    "minor version bump with auto minor upgrade enabled",
			desiredEngineVersion:    "8.0.mysql_aurora.3.10.4",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       false,
		},
		{
			name:                    "minor version bump with auto minor upgrade disabled",
			desiredEngineVersion:    "8.0.mysql_aurora.3.10.4",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: aws.Bool(false),
			expectDifferentAt:       true,
		},
		{
			name:                    "aurora minor version bump with auto minor upgrade enabled",
			desiredEngineVersion:    "8.0.mysql_aurora.3.11.0",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       false,
		},
		{
			name:                    "aurora minor version bump with auto minor upgrade disabled",
			desiredEngineVersion:    "8.0.mysql_aurora.3.11.0",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: aws.Bool(false),
			expectDifferentAt:       true,
		},
		{
			name:                    "aurora major version bump with auto minor upgrade enabled",
			desiredEngineVersion:    "8.0.mysql_aurora.3.10.3",
			latestEngineVersion:     "5.7.mysql_aurora.2.12.0",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       true,
		},
		{
			name:                    "desired short form matches latest major, auto minor upgrade enabled",
			desiredEngineVersion:    "8.0",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       false,
		},
		{
			name:                    "desired short form matches latest major, auto minor upgrade enabled",
			desiredEngineVersion:    "9.0.mysql_aurora.3.10.3",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       true,
		},
		{
			name:                    "desired short form matches latest major, auto minor upgrade disabled",
			desiredEngineVersion:    "8.0",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: aws.Bool(false),
			expectDifferentAt:       true,
		},
		{
			name:                    "desired short form differs from latest major, auto minor upgrade enabled",
			desiredEngineVersion:    "5.7",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       true,
		},
		{
			name:                    "both short form same version",
			desiredEngineVersion:    "8.0",
			latestEngineVersion:     "8.0",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       false,
		},
		{
			name:                    "postgres aurora versions same major",
			desiredEngineVersion:    "15.4",
			latestEngineVersion:     "15.4",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       false,
		},
		{
			name:                    "postgres aurora versions different minor",
			desiredEngineVersion:    "15.5",
			latestEngineVersion:     "15.4",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       false,
		},
		{
			name:                    "postgres aurora versions different major",
			desiredEngineVersion:    "16.1",
			latestEngineVersion:     "15.4",
			autoMinorVersionUpgrade: aws.Bool(true),
			expectDifferentAt:       true,
		},
		{
			name:                    "auto minor upgrade defaults to true when nil",
			desiredEngineVersion:    "8.0.mysql_aurora.3.10.4",
			latestEngineVersion:     "8.0.mysql_aurora.3.10.3",
			autoMinorVersionUpgrade: nil,
			expectDifferentAt:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			desired := &resource{
				ko: &svcapitypes.DBCluster{
					Spec: svcapitypes.DBClusterSpec{
						EngineVersion:           aws.String(tc.desiredEngineVersion),
						AutoMinorVersionUpgrade: tc.autoMinorVersionUpgrade,
					},
				},
			}
			latest := &resource{
				ko: &svcapitypes.DBCluster{
					Spec: svcapitypes.DBClusterSpec{
						EngineVersion: aws.String(tc.latestEngineVersion),
					},
				},
			}

			delta := newResourceDelta(desired, latest)
			assert.Equal(t, tc.expectDifferentAt, delta.DifferentAt("Spec.EngineVersion"),
				"Spec.EngineVersion delta mismatch")
		})
	}
}
