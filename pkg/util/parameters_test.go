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

	"github.com/aws/aws-sdk-go-v2/aws"
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

func TestGetParametersDifference_NilObservedValue(t *testing.T) {
	// RDS reports a parameter left at its engine default with no value as a nil
	// ParameterValue. See aws-controllers-k8s/community#3065.

	from := Parameters{
		"log_statement":          nil,
		"max_slot_wal_keep_size": nil,
	}

	to := Parameters{
		"log_statement":          aws.String("ddl"),
		"max_slot_wal_keep_size": aws.String("2048"),
	}

	added, unchanged, removed := GetParametersDifference(to, from)

	if len(added) != 2 {
		t.Errorf("Expected 2 modified parameters, got %d: %v", len(added), added)
	}

	if len(removed) != 0 {
		t.Errorf("Expected 0 removed parameters, got %d: %v", len(removed), removed)
	}

	if len(unchanged) != 0 {
		t.Errorf("Expected 0 unchanged parameters, got %d: %v", len(unchanged), unchanged)
	}
}

func TestGetParametersDifference_NilOnBothSides(t *testing.T) {
	from := Parameters{
		"log_statement": nil,
	}

	to := Parameters{
		"log_statement": nil,
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

func TestGetParametersDifference_NilDesiredValue(t *testing.T) {
	from := Parameters{
		"log_statement": aws.String("ddl"),
		"work_mem":      aws.String("4096"),
	}

	to := Parameters{
		"log_statement": nil,
	}

	added, unchanged, removed := GetParametersDifference(to, from)

	if len(added) != 1 {
		t.Errorf("Expected 1 modified parameter, got %d: %v", len(added), added)
	}

	if len(removed) != 1 {
		t.Errorf("Expected 1 removed parameter, got %d: %v", len(removed), removed)
	}

	if len(unchanged) != 0 {
		t.Errorf("Expected 0 unchanged parameters, got %d: %v", len(unchanged), unchanged)
	}
}

func TestEqualStringPtr(t *testing.T) {
	tests := []struct {
		name string
		a    *string
		b    *string
		want bool
	}{
		{"both nil", nil, nil, true},
		{"a nil", nil, aws.String("ddl"), false},
		{"b nil", aws.String("ddl"), nil, false},
		{"equal values", aws.String("ddl"), aws.String("ddl"), true},
		{"different values", aws.String("ddl"), aws.String("all"), false},
		{"empty and nil", aws.String(""), nil, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := EqualStringPtr(test.a, test.b); got != test.want {
				t.Errorf("Expected %v, got %v", test.want, got)
			}
		})
	}
}
