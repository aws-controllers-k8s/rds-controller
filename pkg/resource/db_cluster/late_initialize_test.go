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
	"github.com/stretchr/testify/require"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

// observedAuroraServerlessV2 returns the spec DescribeDBClusters reports for
// an Aurora PostgreSQL Serverless v2 cluster created with only an
// identifier, engine and scaling range, i.e. every other field is an RDS
// server-side default.
func observedAuroraServerlessV2() *resource {
	return &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier:        aws.String("adopted-cluster"),
				Engine:                     aws.String("aurora-postgresql"),
				AutoMinorVersionUpgrade:    aws.Bool(true),
				AvailabilityZones:          []*string{aws.String("us-west-2a"), aws.String("us-west-2d"), aws.String("us-west-2b")},
				BackupRetentionPeriod:      aws.Int64(1),
				CopyTagsToSnapshot:         aws.Bool(false),
				DatabaseInsightsMode:       aws.String("standard"),
				DeletionProtection:         aws.Bool(false),
				EngineMode:                 aws.String("provisioned"),
				EngineVersion:              aws.String("16.13"),
				KMSKeyID:                   aws.String("arn:aws:kms:us-west-2:111122223333:key/example"),
				MasterUsername:             aws.String("postgres"),
				NetworkType:                aws.String("IPV4"),
				Port:                       aws.Int64(5432),
				PreferredBackupWindow:      aws.String("06:49-07:19"),
				PreferredMaintenanceWindow: aws.String("mon:11:59-mon:12:29"),
				StorageEncrypted:           aws.Bool(true),
				ServerlessV2ScalingConfiguration: &svcapitypes.ServerlessV2ScalingConfiguration{
					MinCapacity:           aws.Float64(0),
					MaxCapacity:           aws.Float64(4),
					SecondsUntilAutoPause: aws.Int64(300),
				},
			},
		},
	}
}

func TestLateInitializeFromReadOneOutput_AdoptedClusterConverges(t *testing.T) {
	// A cluster adopted with adopt-or-create keeps the user's sparse spec, so
	// every server-side default shows up as a nil-vs-value delta that
	// ModifyDBCluster cannot resolve. Late initialization must copy those
	// defaults into spec so that the next reconcile sees no delta.
	rm := &resourceManager{}
	observed := observedAuroraServerlessV2()
	desired := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier: aws.String("adopted-cluster"),
				Engine:              aws.String("aurora-postgresql"),
				ServerlessV2ScalingConfiguration: &svcapitypes.ServerlessV2ScalingConfiguration{
					MinCapacity: aws.Float64(0),
					MaxCapacity: aws.Float64(4),
				},
			},
		},
	}

	assert.True(t, newResourceDelta(desired, observedAuroraServerlessV2()).DifferentAt("Spec"))

	lateInitialized := rm.concreteResource(rm.lateInitializeFromReadOneOutput(observed, desired))

	assert.Equal(t, observed.ko.Spec, lateInitialized.ko.Spec)
	delta := newResourceDelta(lateInitialized, observedAuroraServerlessV2())
	assert.Empty(t, delta.Differences)
}

func TestLateInitializeFromReadOneOutput_PreservesUserValues(t *testing.T) {
	rm := &resourceManager{}
	observed := observedAuroraServerlessV2()
	desired := &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier:   aws.String("adopted-cluster"),
				Engine:                aws.String("aurora-postgresql"),
				DeletionProtection:    aws.Bool(true),
				PreferredBackupWindow: aws.String("00:00-02:00"),
			},
		},
	}

	lateInitialized := rm.concreteResource(rm.lateInitializeFromReadOneOutput(observed, desired))

	// Values the user set are a real desired change and must not be
	// overwritten by what RDS currently reports.
	assert.Equal(t, aws.Bool(true), lateInitialized.ko.Spec.DeletionProtection)
	assert.Equal(t, aws.String("00:00-02:00"), lateInitialized.ko.Spec.PreferredBackupWindow)
	// A scaling configuration the user left unset stays unmanaged; only its
	// nested SecondsUntilAutoPause is late-initialized, and only when the
	// user manages the parent.
	assert.Nil(t, lateInitialized.ko.Spec.ServerlessV2ScalingConfiguration)
	require.NotNil(t, lateInitialized.ko.Spec.EngineMode)
	assert.Equal(t, "provisioned", *lateInitialized.ko.Spec.EngineMode)
}
