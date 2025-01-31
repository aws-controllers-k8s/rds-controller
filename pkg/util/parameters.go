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
	"errors"
	"fmt"

	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
)

var (
	ErrUnknownParameter      = fmt.Errorf("unknown parameter")
	ErrUnmodifiableParameter = fmt.Errorf("parameter is not modifiable")
)

// Parameters represents the elements of a DB Parameter Group
// or a DB Cluster Parameter Group
type Parameters map[string]*string

// NewErrUnknownParameter generates a NotFound error about
// an unknown parameter
func NewErrUnknownParameter(name string) error {
	return NotFound(
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

// GetParametersDifference compares two Parameters maps and returns the
// parameters to add & update, the unchanged parameters, and
// the parameters to remove. It also handles the case where parameters
// that were previously invalid are removed.
func GetParametersDifference(
	to, from Parameters,
) (added, unchanged, removed Parameters) {
	added = Parameters{}
	unchanged = Parameters{}
	removed = Parameters{}

	// Find added and unchanged parameters
	for toKey, toVal := range to {
		if fromVal, exists := from[toKey]; exists {
			// Parameter exists in both maps
			if (toVal == nil && fromVal == nil) || (toVal != nil && fromVal != nil && *toVal == *fromVal) {
				unchanged[toKey] = toVal
			} else {
				added[toKey] = toVal // Different values = modified parameter
			}
		} else {
			added[toKey] = toVal // Not in 'from' = new parameter
		}
	}

	// Find removed parameters
	for fromKey, fromVal := range from {
		if _, exists := to[fromKey]; !exists {
			// If a parameter is removed and it was previously causing an error
			// (i.e., it was invalid), we still mark it as removed but don't
			// need to take any special action since removing an invalid parameter
			// is the desired behavior
			removed[fromKey] = fromVal
		}
	}

	return added, unchanged, removed
}

// ChunkParameters splits a supplied map of parameters into multiple
// slices of maps of parameters of a given size.
func ChunkParameters(
	input Parameters,
	chunkSize int,
) []Parameters {
	var chunks []Parameters
	chunk := Parameters{}
	idx := 0
	for k, v := range input {
		if idx < chunkSize {
			chunk[k] = v
			idx++
		} else {
			// reset the chunker
			chunks = append(chunks, chunk)
			chunk = Parameters{}
			idx = 0
		}
	}
	chunks = append(chunks, chunk)

	return chunks
}

// IsNotFound returns true if the supplied error is a NotFound error
func IsNotFound(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

// NotFound returns a NotFoundError with the supplied error
func NotFound(err error) error {
	return &NotFoundError{err}
}

// NotFoundError represents a type of error when a requested resource is not
// found
type NotFoundError struct {
	err error
}

func (e *NotFoundError) Error() string {
	return e.err.Error()
}
