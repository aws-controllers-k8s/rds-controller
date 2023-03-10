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
	"fmt"

	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
)

var (
	ErrUnknownParameter      = fmt.Errorf("unknown parameter")
	ErrUnmodifiableParameter = fmt.Errorf("parameter is not modifiable")
)

// NewErrUnknownParameter generates an ACK terminal error about
// an unknown parameter
func NewErrUnknownParameter(name string) error {
	// This is a terminal error because unless the user removes this parameter
	// from their list of parameter overrides, we will not be able to get the
	// resource into a synced state.
	return ackerr.NewTerminalError(
		fmt.Errorf("%w: %s", ErrUnknownParameter, name),
	)
}

// NewErrUnmodifiableParameter generates an ACK terminal error about
// a parameter that may not be modified
func NewErrUnmodifiableParameter(name string) error {
	// This is a terminal error because unless the user removes this parameter
	// from their list of parameter overrides, we will not be able to get the
	// resource into a synced state.
	return ackerr.NewTerminalError(
		fmt.Errorf("%w: %s", ErrUnmodifiableParameter, name),
	)
}

// ComputeParametersDelta compares two Parameter arrays and returns the new
// parameters to add, to update and the parameter identifiers to delete
func ComputeParametersDelta(
	desired map[string]*string,
	latest map[string]*string,
) (map[string]*string, []string) {
	toReset := []string{}
	toModify := map[string]*string{}

	for k, v := range desired {
		if lv, found := latest[k]; !found {
			toModify[k] = v
		} else if !equalStrings(v, lv) {
			toModify[k] = v
		}
	}
	for k := range latest {
		if _, found := desired[k]; !found {
			toReset = append(toReset, k)
		}
	}
	return toModify, toReset
}

// mapStringChunks splits a supplied map of string pointers into multiple
// slices of maps of string pointers of a given size.
func MapStringChunks(
	input map[string]*string,
	chunkSize int,
) []map[string]*string {
	var chunks []map[string]*string
	chunk := map[string]*string{}
	idx := 0
	for k, v := range input {
		if idx < chunkSize {
			chunk[k] = v
			idx++
		} else {
			// reset the chunker
			chunks = append(chunks, chunk)
			chunk = map[string]*string{}
			idx = 0
		}
	}
	chunks = append(chunks, chunk)

	return chunks
}
