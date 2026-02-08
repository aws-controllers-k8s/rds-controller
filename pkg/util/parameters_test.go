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
	"testing"

	"github.com/aws/aws-sdk-go/aws"
)

func TestGetParametersDifference_PointerComparison(t *testing.T) {
	value1 := aws.String("ROW")
	value2 := aws.String("ROW")

	from := Parameters{
		"binlog_format": value1,
	}

	to := Parameters{
		"binlog_format": value2,
	}

	added, unchanged, removed := GetParametersDifference(to, from)

	if len(added) != 0 {
		t.Errorf("Expected 0 modified parameters, got %d: %v", len(added), added)
	}

	if len(removed) != 0 {
		t.Errorf("Expected 0 removed parameters, got %d: %v", len(removed), removed)
	}

	if len(unchanged) != 1 {
		t.Errorf("Expected 1 unchanged parameter, got %d: %v", len(unchanged), unchanged)
	}
}

func TestGetParametersDifference_ActualDifference(t *testing.T) {
	// Test that actual differences are correctly detected

	from := Parameters{
		"binlog_format":   aws.String("OFF"),
		"max_connections": aws.String("100"),
	}

	to := Parameters{
		"binlog_format":   aws.String("ROW"),
		"max_connections": aws.String("100"),
	}

	added, unchanged, removed := GetParametersDifference(to, from)

	// binlog_format changed from OFF to ROW, so should be in "added" (modify)
	// max_connections stayed the same, so should be in "unchanged"
	// Nothing was removed (no parameters absent from 'to')

	if len(added) != 1 {
		t.Errorf("Expected 1 modified parameter, got %d: %v", len(added), added)
	}

	if len(removed) != 0 {
		t.Errorf("Expected 0 removed parameters, got %d: %v", len(removed), removed)
	}

	if len(unchanged) != 1 {
		t.Errorf("Expected 1 unchanged parameter, got %d: %v", len(unchanged), unchanged)
	}
}

func TestGetParametersDifference_NewParameter(t *testing.T) {
	from := Parameters{
		"max_connections": aws.String("100"),
	}

	to := Parameters{
		"max_connections": aws.String("100"),
		"binlog_format":   aws.String("ROW"),
	}

	added, unchanged, removed := GetParametersDifference(to, from)

	if len(added) != 1 {
		t.Errorf("Expected 1 modified parameter, got %d: %v", len(added), added)
	}

	if len(removed) != 0 {
		t.Errorf("Expected 0 removed parameters, got %d: %v", len(removed), removed)
	}

	if len(unchanged) != 1 {
		t.Errorf("Expected 1 unchanged parameter, got %d: %v", len(unchanged), unchanged)
	}
}

func TestGetParametersDifference_RemoveParameter(t *testing.T) {
	from := Parameters{
		"max_connections": aws.String("100"),
		"binlog_format":   aws.String("ROW"),
	}

	to := Parameters{
		"max_connections": aws.String("100"),
	}

	added, unchanged, removed := GetParametersDifference(to, from)

	if len(added) != 0 {
		t.Errorf("Expected 0 modified parameters, got %d: %v", len(added), added)
	}

	if len(removed) != 1 {
		t.Errorf("Expected 1 removed parameter, got %d: %v", len(removed), removed)
	}

	if len(unchanged) != 1 {
		t.Errorf("Expected 1 unchanged parameter, got %d: %v", len(unchanged), unchanged)
	}
}
