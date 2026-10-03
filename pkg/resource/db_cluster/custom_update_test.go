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

func TestNewUpdateRequestPayload_PreferredBackupWindow(t *testing.T) {
	// PreferredBackupWindow is included in the ModifyDBCluster API call when
	// Spec.PreferredBackupWindow is in the delta.
	rm := &resourceManager{}
	ctx := context.Background()

	desired := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier:   aws.String("test-cluster"),
				PreferredBackupWindow: aws.String("07:00-09:00"),
			},
		},
	}

	delta := ackcompare.NewDelta()
	delta.Add("Spec.PreferredBackupWindow", aws.String("05:00-07:00"), desired.ko.Spec.PreferredBackupWindow)

	input, err := rm.newUpdateRequestPayload(ctx, desired, delta)

	assert.NoError(t, err)
	assert.NotNil(t, input)
	assert.Equal(t, *desired.ko.Spec.PreferredBackupWindow, *input.PreferredBackupWindow)
}

func TestNewUpdateRequestPayload_PreferredBackupWindowNotInDelta(t *testing.T) {
	// PreferredBackupWindow is NOT included in the ModifyDBCluster API call when
	// Spec.PreferredBackupWindow is NOT in the delta.
	rm := &resourceManager{}
	ctx := context.Background()

	desired := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier:   aws.String("test-cluster"),
				PreferredBackupWindow: aws.String("07:00-09:00"),
			},
		},
	}

	delta := ackcompare.NewDelta()

	input, err := rm.newUpdateRequestPayload(ctx, desired, delta)

	assert.NoError(t, err)
	assert.NotNil(t, input)
	assert.Nil(t, input.PreferredBackupWindow)
}

// TestNewUpdateRequestPayload_ServerlessV2ScalingConfiguration_FirstTimeAdd is
// the regression test for aws-controllers-k8s/community#3036. Adding
// ServerlessV2ScalingConfiguration to a DBCluster that previously had none
// (nil->populated) must produce a ModifyDBClusterInput whose
// ServerlessV2ScalingConfiguration carries MinCapacity/MaxCapacity, so the
// change actually reaches AWS.
func TestNewUpdateRequestPayload_ServerlessV2ScalingConfiguration_FirstTimeAdd(t *testing.T) {
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

	input, err := rm.newUpdateRequestPayload(ctx, desired, delta)
	assert.NoError(err)
	assert.NotNil(input)

	// min/max must be carried into the modify input so the change converges.
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

// TestNewUpdateRequestPayload_ServerlessV2ScalingConfiguration_SecondsUntilAutoPause
// asserts that SecondsUntilAutoPause flows through the generated payload when
// the SV2SC parent differs.
func TestNewUpdateRequestPayload_ServerlessV2ScalingConfiguration_SecondsUntilAutoPause(t *testing.T) {
	assert := assert.New(t)

	rm := &resourceManager{}
	ctx := context.Background()

	desired := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier: aws.String("test-cluster"),
				ServerlessV2ScalingConfiguration: &svcapitypes.ServerlessV2ScalingConfiguration{
					MinCapacity:           aws.Float64(0.5),
					MaxCapacity:           aws.Float64(4),
					SecondsUntilAutoPause: aws.Int64(3600),
				},
			},
		},
	}

	latest := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier: aws.String("test-cluster"),
				ServerlessV2ScalingConfiguration: &svcapitypes.ServerlessV2ScalingConfiguration{
					MinCapacity:           aws.Float64(0.5),
					MaxCapacity:           aws.Float64(2),
					SecondsUntilAutoPause: aws.Int64(300),
				},
			},
		},
	}

	delta := newResourceDelta(desired, latest)
	assert.True(
		delta.DifferentAt("Spec.ServerlessV2ScalingConfiguration"),
		"expected Spec.ServerlessV2ScalingConfiguration to differ",
	)

	input, err := rm.newUpdateRequestPayload(ctx, desired, delta)
	assert.NoError(err)
	assert.NotNil(input)
	assert.NotNil(input.ServerlessV2ScalingConfiguration)
	assert.NotNil(
		input.ServerlessV2ScalingConfiguration.SecondsUntilAutoPause,
		"SecondsUntilAutoPause must flow through the generated payload when SV2SC differs",
	)
	assert.Equal(
		int32(*desired.ko.Spec.ServerlessV2ScalingConfiguration.SecondsUntilAutoPause),
		*input.ServerlessV2ScalingConfiguration.SecondsUntilAutoPause,
	)
}
