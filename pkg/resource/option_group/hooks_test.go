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

package option_group

import (
	"testing"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

func setting(name, value string) *svcapitypes.OptionSetting {
	return &svcapitypes.OptionSetting{Name: aws.String(name), Value: aws.String(value)}
}

func optionRes(options []*svcapitypes.OptionConfiguration) *resource {
	return &resource{
		ko: &svcapitypes.OptionGroup{
			Spec: svcapitypes.OptionGroupSpec{Options: options},
		},
	}
}

func TestCompareOptions(t *testing.T) {
	tests := []struct {
		name              string
		desired           []*svcapitypes.OptionConfiguration
		latest            []*svcapitypes.OptionConfiguration
		expectDifferentAt bool
	}{
		{
			name:              "both empty",
			desired:           nil,
			latest:            nil,
			expectDifferentAt: false,
		},
		{
			name: "observed carries extra default settings the user did not specify",
			desired: []*svcapitypes.OptionConfiguration{
				{
					OptionName:     aws.String("MARIADB_AUDIT_PLUGIN"),
					OptionSettings: []*svcapitypes.OptionSetting{setting("SERVER_AUDIT_EVENTS", "CONNECT")},
				},
			},
			latest: []*svcapitypes.OptionConfiguration{
				{
					OptionName: aws.String("MARIADB_AUDIT_PLUGIN"),
					OptionSettings: []*svcapitypes.OptionSetting{
						setting("SERVER_AUDIT_EVENTS", "CONNECT"),
						setting("SERVER_AUDIT_FILE_ROTATE_SIZE", "1000000"),
					},
				},
			},
			expectDifferentAt: false,
		},
		{
			name: "desired setting value differs from observed",
			desired: []*svcapitypes.OptionConfiguration{
				{
					OptionName:     aws.String("MARIADB_AUDIT_PLUGIN"),
					OptionSettings: []*svcapitypes.OptionSetting{setting("SERVER_AUDIT_EVENTS", "CONNECT,QUERY")},
				},
			},
			latest: []*svcapitypes.OptionConfiguration{
				{
					OptionName:     aws.String("MARIADB_AUDIT_PLUGIN"),
					OptionSettings: []*svcapitypes.OptionSetting{setting("SERVER_AUDIT_EVENTS", "CONNECT")},
				},
			},
			expectDifferentAt: true,
		},
		{
			name: "desired option missing from observed",
			desired: []*svcapitypes.OptionConfiguration{
				{OptionName: aws.String("MARIADB_AUDIT_PLUGIN")},
			},
			latest:            []*svcapitypes.OptionConfiguration{},
			expectDifferentAt: true,
		},
		{
			name:    "observed option no longer desired",
			desired: []*svcapitypes.OptionConfiguration{},
			latest: []*svcapitypes.OptionConfiguration{
				{OptionName: aws.String("MARIADB_AUDIT_PLUGIN")},
			},
			expectDifferentAt: true,
		},
		{
			name: "same options in different order",
			desired: []*svcapitypes.OptionConfiguration{
				{OptionName: aws.String("OPTION_A")},
				{OptionName: aws.String("OPTION_B")},
			},
			latest: []*svcapitypes.OptionConfiguration{
				{OptionName: aws.String("OPTION_B")},
				{OptionName: aws.String("OPTION_A")},
			},
			expectDifferentAt: false,
		},
		{
			name: "desired requires setting absent from observed",
			desired: []*svcapitypes.OptionConfiguration{
				{
					OptionName:     aws.String("MARIADB_AUDIT_PLUGIN"),
					OptionSettings: []*svcapitypes.OptionSetting{setting("SERVER_AUDIT_EVENTS", "CONNECT")},
				},
			},
			latest: []*svcapitypes.OptionConfiguration{
				{OptionName: aws.String("MARIADB_AUDIT_PLUGIN")},
			},
			expectDifferentAt: true,
		},
		{
			name: "desired version differs from observed",
			desired: []*svcapitypes.OptionConfiguration{
				{OptionName: aws.String("OEM_AGENT"), OptionVersion: aws.String("13.5.0.0.v1")},
			},
			latest: []*svcapitypes.OptionConfiguration{
				{OptionName: aws.String("OEM_AGENT"), OptionVersion: aws.String("13.4.0.0.v1")},
			},
			expectDifferentAt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta := ackcompare.NewDelta()
			compareOptions(delta, optionRes(tt.desired), optionRes(tt.latest))
			assert.Equal(t, tt.expectDifferentAt, delta.DifferentAt("Spec.Options"))
		})
	}
}

func TestOptionsToRemove(t *testing.T) {
	desired := []*svcapitypes.OptionConfiguration{
		{OptionName: aws.String("KEEP")},
	}
	latest := []*svcapitypes.OptionConfiguration{
		{OptionName: aws.String("KEEP")},
		{OptionName: aws.String("DROP")},
	}
	assert.Equal(t, []string{"DROP"}, optionsToRemove(desired, latest))
	assert.Empty(t, optionsToRemove(desired, nil))
}

func TestOptionConfigurationsFromObserved(t *testing.T) {
	observed := []*svcapitypes.Option{
		{
			OptionName:    aws.String("MARIADB_AUDIT_PLUGIN"),
			OptionVersion: aws.String("1.1"),
			Port:          aws.Int64(1158),
			DBSecurityGroupMemberships: []*svcapitypes.DBSecurityGroupMembership{
				{DBSecurityGroupName: aws.String("sg-1")},
			},
			VPCSecurityGroupMemberships: []*svcapitypes.VPCSecurityGroupMembership{
				{VPCSecurityGroupID: aws.String("vpc-sg-1")},
			},
			OptionSettings: []*svcapitypes.OptionSetting{setting("SERVER_AUDIT_EVENTS", "CONNECT")},
		},
	}
	got := optionConfigurationsFromObserved(observed)
	assert.Len(t, got, 1)
	assert.Equal(t, "MARIADB_AUDIT_PLUGIN", *got[0].OptionName)
	assert.Equal(t, "1.1", *got[0].OptionVersion)
	assert.Equal(t, int64(1158), *got[0].Port)
	assert.Equal(t, []*string{aws.String("sg-1")}, got[0].DBSecurityGroupMemberships)
	assert.Equal(t, []*string{aws.String("vpc-sg-1")}, got[0].VPCSecurityGroupMemberships)
	assert.Len(t, got[0].OptionSettings, 1)
	assert.Nil(t, optionConfigurationsFromObserved(nil))
}

func TestSDKOptionConfigurationsFromResource(t *testing.T) {
	options := []*svcapitypes.OptionConfiguration{
		{
			OptionName:                  aws.String("MARIADB_AUDIT_PLUGIN"),
			Port:                        aws.Int64(1158),
			DBSecurityGroupMemberships:  []*string{aws.String("sg-1")},
			VPCSecurityGroupMemberships: []*string{aws.String("vpc-sg-1")},
			OptionSettings:              []*svcapitypes.OptionSetting{setting("SERVER_AUDIT_EVENTS", "CONNECT")},
		},
	}
	got := sdkOptionConfigurationsFromResource(options)
	assert.Len(t, got, 1)
	assert.Equal(t, "MARIADB_AUDIT_PLUGIN", *got[0].OptionName)
	assert.Equal(t, int32(1158), *got[0].Port)
	assert.Equal(t, []string{"sg-1"}, got[0].DBSecurityGroupMemberships)
	assert.Equal(t, []string{"vpc-sg-1"}, got[0].VpcSecurityGroupMemberships)
	assert.Len(t, got[0].OptionSettings, 1)
	assert.Equal(t, "SERVER_AUDIT_EVENTS", *got[0].OptionSettings[0].Name)
	assert.Nil(t, sdkOptionConfigurationsFromResource(nil))
}
