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

// TestNewResourceDelta_CustomPreCompare_DBClusterIdentifierNil validates that
// when DBClusterIdentifier is nil, the fields guarded by the
// `if a.ko.Spec.DBClusterIdentifier == nil` block in customPreCompare are
// included in the Delta returned by newResourceDelta.
func TestNewResourceDelta_CustomPreCompare_DBClusterIdentifierNil(t *testing.T) {
	tests := []struct {
		name      string
		fieldPath string
		desired   func() *svcapitypes.DBInstanceSpec
		latest    func() *svcapitypes.DBInstanceSpec
	}{
		{
			name:      "StorageType difference is detected",
			fieldPath: "Spec.StorageType",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					StorageType: aws.String("gp3"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					StorageType: aws.String("gp2"),
				}
			},
		},
		{
			name:      "DatabaseInsightsMode difference is detected",
			fieldPath: "Spec.DatabaseInsightsMode",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DatabaseInsightsMode: aws.String("advanced"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DatabaseInsightsMode: aws.String("standard"),
				}
			},
		},
		{
			name:      "EnableCloudwatchLogsExports difference is detected",
			fieldPath: "Spec.EnableCloudwatchLogsExports",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					EnableCloudwatchLogsExports: []*string{aws.String("audit"), aws.String("error")},
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					EnableCloudwatchLogsExports: []*string{aws.String("audit")},
				}
			},
		},
		{
			name:      "MaxAllocatedStorage difference is detected",
			fieldPath: "Spec.MaxAllocatedStorage",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					MaxAllocatedStorage: aws.Int64(200),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					MaxAllocatedStorage: aws.Int64(100),
				}
			},
		},
		{
			name:      "BackupRetentionPeriod difference is detected",
			fieldPath: "Spec.BackupRetentionPeriod",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					BackupRetentionPeriod: aws.Int64(14),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					BackupRetentionPeriod: aws.Int64(7),
				}
			},
		},
		{
			name:      "PreferredBackupWindow difference is detected",
			fieldPath: "Spec.PreferredBackupWindow",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					PreferredBackupWindow: aws.String("07:00-09:00"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					PreferredBackupWindow: aws.String("03:00-05:00"),
				}
			},
		},
		{
			name:      "DeletionProtection difference is detected",
			fieldPath: "Spec.DeletionProtection",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DeletionProtection: aws.Bool(true),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DeletionProtection: aws.Bool(false),
				}
			},
		},
		{
			name:      "EngineVersion difference is detected",
			fieldPath: "Spec.EngineVersion",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					EngineVersion: aws.String("15.4"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					EngineVersion: aws.String("14.9"),
				}
			},
		},
		{
			name:      "MasterUsername difference is detected",
			fieldPath: "Spec.MasterUsername",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					MasterUsername: aws.String("admin"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					MasterUsername: aws.String("root"),
				}
			},
		},
		{
			name:      "DBName difference is detected",
			fieldPath: "Spec.DBName",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBName: aws.String("mydb"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBName: aws.String("otherdb"),
				}
			},
		},
		{
			name:      "PerformanceInsightsEnabled difference is detected",
			fieldPath: "Spec.PerformanceInsightsEnabled",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					PerformanceInsightsEnabled: aws.Bool(true),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					PerformanceInsightsEnabled: aws.Bool(false),
				}
			},
		},
		{
			name:      "PerformanceInsightsKMSKeyID difference is detected",
			fieldPath: "Spec.PerformanceInsightsKMSKeyID",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					PerformanceInsightsKMSKeyID: aws.String("arn:aws:kms:us-east-1:123456789012:key/new-key"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					PerformanceInsightsKMSKeyID: aws.String("arn:aws:kms:us-east-1:123456789012:key/old-key"),
				}
			},
		},
		{
			name:      "PerformanceInsightsRetentionPeriod difference is detected",
			fieldPath: "Spec.PerformanceInsightsRetentionPeriod",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					PerformanceInsightsRetentionPeriod: aws.Int64(31),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					PerformanceInsightsRetentionPeriod: aws.Int64(7),
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			desired := &resource{
				ko: &svcapitypes.DBInstance{
					Spec: *tc.desired(),
				},
			}
			latest := &resource{
				ko: &svcapitypes.DBInstance{
					Spec: *tc.latest(),
				},
			}
			// DBClusterIdentifier is nil (default) so the fields should be compared
			delta := newResourceDelta(desired, latest)
			assert.True(t, delta.DifferentAt(tc.fieldPath),
				"expected delta to contain difference at %s when DBClusterIdentifier is nil", tc.fieldPath)
		})
	}
}

// TestNewResourceDelta_CustomPreCompare_DBClusterIdentifierSet validates that
// when DBClusterIdentifier is set, the fields guarded by the
// `if a.ko.Spec.DBClusterIdentifier == nil` block in customPreCompare are
// NOT included in the Delta returned by newResourceDelta.
func TestNewResourceDelta_CustomPreCompare_DBClusterIdentifierSet(t *testing.T) {
	tests := []struct {
		name      string
		fieldPath string
		desired   func() *svcapitypes.DBInstanceSpec
		latest    func() *svcapitypes.DBInstanceSpec
	}{
		{
			name:      "StorageType difference is suppressed",
			fieldPath: "Spec.StorageType",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier: aws.String("my-cluster"),
					StorageType:         aws.String("gp3"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier: aws.String("my-cluster"),
					StorageType:         aws.String("gp2"),
				}
			},
		},
		{
			name:      "DatabaseInsightsMode difference is suppressed",
			fieldPath: "Spec.DatabaseInsightsMode",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:  aws.String("my-cluster"),
					DatabaseInsightsMode: aws.String("advanced"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:  aws.String("my-cluster"),
					DatabaseInsightsMode: aws.String("standard"),
				}
			},
		},
		{
			name:      "EnableCloudwatchLogsExports difference is suppressed",
			fieldPath: "Spec.EnableCloudwatchLogsExports",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:         aws.String("my-cluster"),
					EnableCloudwatchLogsExports: []*string{aws.String("audit"), aws.String("error")},
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:         aws.String("my-cluster"),
					EnableCloudwatchLogsExports: []*string{aws.String("audit")},
				}
			},
		},
		{
			name:      "BackupRetentionPeriod difference is suppressed",
			fieldPath: "Spec.BackupRetentionPeriod",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:   aws.String("my-cluster"),
					BackupRetentionPeriod: aws.Int64(14),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:   aws.String("my-cluster"),
					BackupRetentionPeriod: aws.Int64(7),
				}
			},
		},
		{
			name:      "DeletionProtection difference is suppressed",
			fieldPath: "Spec.DeletionProtection",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier: aws.String("my-cluster"),
					DeletionProtection:  aws.Bool(true),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier: aws.String("my-cluster"),
					DeletionProtection:  aws.Bool(false),
				}
			},
		},
		{
			name:      "EngineVersion difference is suppressed",
			fieldPath: "Spec.EngineVersion",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier: aws.String("my-cluster"),
					EngineVersion:       aws.String("15.4"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier: aws.String("my-cluster"),
					EngineVersion:       aws.String("14.9"),
				}
			},
		},
		{
			name:      "MasterUsername difference is suppressed",
			fieldPath: "Spec.MasterUsername",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier: aws.String("my-cluster"),
					MasterUsername:      aws.String("admin"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier: aws.String("my-cluster"),
					MasterUsername:      aws.String("root"),
				}
			},
		},
		{
			name:      "DBName difference is suppressed",
			fieldPath: "Spec.DBName",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier: aws.String("my-cluster"),
					DBName:              aws.String("mydb"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier: aws.String("my-cluster"),
					DBName:              aws.String("otherdb"),
				}
			},
		},
		{
			name:      "PerformanceInsightsEnabled difference is suppressed",
			fieldPath: "Spec.PerformanceInsightsEnabled",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:        aws.String("my-cluster"),
					PerformanceInsightsEnabled: aws.Bool(true),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:        aws.String("my-cluster"),
					PerformanceInsightsEnabled: aws.Bool(false),
				}
			},
		},
		{
			name:      "PerformanceInsightsKMSKeyID difference is suppressed",
			fieldPath: "Spec.PerformanceInsightsKMSKeyID",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:         aws.String("my-cluster"),
					PerformanceInsightsKMSKeyID: aws.String("arn:aws:kms:us-east-1:123456789012:key/new-key"),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:         aws.String("my-cluster"),
					PerformanceInsightsKMSKeyID: aws.String("arn:aws:kms:us-east-1:123456789012:key/old-key"),
				}
			},
		},
		{
			name:      "PerformanceInsightsRetentionPeriod difference is suppressed",
			fieldPath: "Spec.PerformanceInsightsRetentionPeriod",
			desired: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:                aws.String("my-cluster"),
					PerformanceInsightsRetentionPeriod: aws.Int64(31),
				}
			},
			latest: func() *svcapitypes.DBInstanceSpec {
				return &svcapitypes.DBInstanceSpec{
					DBClusterIdentifier:                aws.String("my-cluster"),
					PerformanceInsightsRetentionPeriod: aws.Int64(7),
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			desired := &resource{
				ko: &svcapitypes.DBInstance{
					Spec: *tc.desired(),
				},
			}
			latest := &resource{
				ko: &svcapitypes.DBInstance{
					Spec: *tc.latest(),
				},
			}
			// DBClusterIdentifier is set so these fields should NOT be compared
			delta := newResourceDelta(desired, latest)
			assert.False(t, delta.DifferentAt(tc.fieldPath),
				"expected delta to NOT contain difference at %s when DBClusterIdentifier is set", tc.fieldPath)
		})
	}
}
