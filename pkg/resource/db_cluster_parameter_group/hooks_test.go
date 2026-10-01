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

package db_cluster_parameter_group

import (
	"testing"

	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go/aws"
)

func TestNewParamMeta(t *testing.T) {
	tests := []struct {
		name             string
		param            svcsdktypes.Parameter
		wantIsModifiable bool
		wantIsDynamic    bool
	}{
		{
			name: "dynamic and modifiable",
			param: svcsdktypes.Parameter{
				ParameterName: aws.String("log_statement"),
				IsModifiable:  aws.Bool(true),
				ApplyType:     aws.String("dynamic"),
			},
			wantIsModifiable: true,
			wantIsDynamic:    true,
		},
		{
			name: "static is not dynamic",
			param: svcsdktypes.Parameter{
				ParameterName: aws.String("shared_preload_libraries"),
				IsModifiable:  aws.Bool(true),
				ApplyType:     aws.String("static"),
			},
			wantIsModifiable: true,
			wantIsDynamic:    false,
		},
		{
			// Absent means unknown, so stay modifiable and let RDS adjudicate.
			name: "nil IsModifiable does not block modification",
			param: svcsdktypes.Parameter{
				ParameterName: aws.String("log_statement"),
				ApplyType:     aws.String("dynamic"),
			},
			wantIsModifiable: true,
			wantIsDynamic:    true,
		},
		{
			name: "explicitly unmodifiable",
			param: svcsdktypes.Parameter{
				ParameterName: aws.String("rds.extensions"),
				IsModifiable:  aws.Bool(false),
				ApplyType:     aws.String("static"),
			},
			wantIsModifiable: false,
			wantIsDynamic:    false,
		},
		{
			name: "nil ApplyType is not dynamic",
			param: svcsdktypes.Parameter{
				ParameterName: aws.String("log_statement"),
				IsModifiable:  aws.Bool(true),
			},
			wantIsModifiable: true,
			wantIsDynamic:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			meta := newParamMeta(test.param)
			if meta.IsModifiable != test.wantIsModifiable {
				t.Errorf("Expected IsModifiable %v, got %v", test.wantIsModifiable, meta.IsModifiable)
			}
			if meta.IsDynamic != test.wantIsDynamic {
				t.Errorf("Expected IsDynamic %v, got %v", test.wantIsDynamic, meta.IsDynamic)
			}
		})
	}
}
