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
	"testing"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

func TestNewCustomUpdateRequestPayload_PreferredBackupWindow(t *testing.T) {
	// Test case to verify that PreferredBackupWindow is included in the ModifyDBCluster API call
	// when Spec.PreferredBackupWindow is in the delta

	// Setup
	rm := &resourceManager{}
	ctx := context.Background()

	// Create desired resource with PreferredBackupWindow set
	desired := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier:   aws.String("test-cluster"),
				PreferredBackupWindow: aws.String("07:00-09:00"),
			},
		},
	}

	// Create latest resource with different PreferredBackupWindow
	latest := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier:   aws.String("test-cluster"),
				PreferredBackupWindow: aws.String("05:00-07:00"),
			},
		},
	}

	// Create delta with PreferredBackupWindow difference
	delta := ackcompare.NewDelta()
	delta.Add("Spec.PreferredBackupWindow", latest.ko.Spec.PreferredBackupWindow, desired.ko.Spec.PreferredBackupWindow)

	// Call the function under test
	input, err := rm.newCustomUpdateRequestPayload(ctx, desired, latest, delta)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, input)
	assert.Equal(t, *desired.ko.Spec.PreferredBackupWindow, *input.PreferredBackupWindow)
}

func TestNewCustomUpdateRequestPayload_PreferredBackupWindowNotInDelta(t *testing.T) {
	// Test case to verify that PreferredBackupWindow is NOT included in the ModifyDBCluster API call
	// when Spec.PreferredBackupWindow is NOT in the delta

	// Setup
	rm := &resourceManager{}
	ctx := context.Background()

	// Create desired resource with PreferredBackupWindow set
	desired := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier:   aws.String("test-cluster"),
				PreferredBackupWindow: aws.String("07:00-09:00"),
			},
		},
	}

	// Create latest resource with same PreferredBackupWindow
	latest := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier:   aws.String("test-cluster"),
				PreferredBackupWindow: aws.String("07:00-09:00"),
			},
		},
	}

	// Create delta without PreferredBackupWindow difference
	delta := ackcompare.NewDelta()

	// Call the function under test
	input, err := rm.newCustomUpdateRequestPayload(ctx, desired, latest, delta)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, input)
	assert.Nil(t, input.PreferredBackupWindow)
}

// TestNewCustomUpdateRequestPayload_ServerlessV2ScalingConfiguration_FirstTimeAdd
// is a regression test for aws-controllers-k8s/community#3036. Adding
// ServerlessV2ScalingConfiguration to a DBCluster that previously had none
// (nil->populated) must produce a ModifyDBClusterInput whose
// ServerlessV2ScalingConfiguration carries MinCapacity/MaxCapacity, so the
// change actually reaches AWS. This drives the REAL newResourceDelta and the
// REAL newCustomUpdateRequestPayload (no AWS, no mocks). It FAILS before the
// fix (empty struct sent) and PASSES after.
func TestNewCustomUpdateRequestPayload_ServerlessV2ScalingConfiguration_FirstTimeAdd(t *testing.T) {
	assert := assert.New(t)

	rm := &resourceManager{}
	ctx := context.Background()

	// desired: cluster spec now includes SV2SC {MinCapacity:0.5, MaxCapacity:2}
	desired := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier: aws.String("test-cluster"),
				ServerlessV2ScalingConfiguration: &svcapitypes.ServerlessV2ScalingConfiguration{
					MinCapacity: aws.Float64(0.5),
					MaxCapacity: aws.Float64(2),
				},
			},
		},
	}

	// latest: observed cluster has NO SV2SC (nil).
	latest := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier: aws.String("test-cluster"),
			},
		},
	}

	// Use the REAL delta computation, not a hand-built delta.
	delta := newResourceDelta(desired, latest)

	// The parent path must be flagged different (nil->populated).
	assert.True(
		delta.DifferentAt("Spec.ServerlessV2ScalingConfiguration"),
		"expected parent Spec.ServerlessV2ScalingConfiguration to differ on nil->populated",
	)

	// Call the function under test.
	input, err := rm.newCustomUpdateRequestPayload(ctx, desired, latest, delta)
	assert.NoError(err)
	assert.NotNil(input)

	// POST-FIX expectation: min/max must be carried into the modify input so
	// the change actually converges at AWS.
	assert.NotNil(
		input.ServerlessV2ScalingConfiguration,
		"ServerlessV2ScalingConfiguration must be attached to ModifyDBClusterInput",
	)
	assert.NotNil(
		input.ServerlessV2ScalingConfiguration.MaxCapacity,
		"MaxCapacity must be set on first-time add (community#3036)",
	)
	assert.NotNil(
		input.ServerlessV2ScalingConfiguration.MinCapacity,
		"MinCapacity must be set on first-time add (community#3036)",
	)
	assert.Equal(*desired.ko.Spec.ServerlessV2ScalingConfiguration.MaxCapacity, *input.ServerlessV2ScalingConfiguration.MaxCapacity)
	assert.Equal(*desired.ko.Spec.ServerlessV2ScalingConfiguration.MinCapacity, *input.ServerlessV2ScalingConfiguration.MinCapacity)
}
