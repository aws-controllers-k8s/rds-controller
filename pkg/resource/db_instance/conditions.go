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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
)

// getSyncedCondition returns the Condition in the resource's Conditions
// collection that is of type ConditionTypeResourceSynced. If no such condition
// is found, returns nil.
//
// TODO(jaypipes): Move to ACK code-gen templates.
func getSyncedCondition(r *resource) *ackv1alpha1.Condition {
	return getConditionOfType(r, ackv1alpha1.ConditionTypeResourceSynced)
}

// getTerminalCondition returns the Condition in the resource's Conditions
// collection that is of type ConditionTypeTerminal. If no such condition is
// found, returns nil.
//
// TODO(jaypipes): Move to ACK code-gen templates.
func getTerminalCondition(r *resource) *ackv1alpha1.Condition {
	return getConditionOfType(r, ackv1alpha1.ConditionTypeTerminal)
}

// getConditionOfType returns the Condition in the resource's Conditions
// collection of the supplied type. If no such condition is found, returns nil.
//
// TODO(jaypipes): Move to ACK code-gen templates.
func getConditionOfType(
	r *resource,
	condType ackv1alpha1.ConditionType,
) *ackv1alpha1.Condition {
	for _, condition := range r.ko.Status.Conditions {
		if condition.Type == condType {
			return condition
		}
	}
	return nil
}

// setSyncedCondition sets the resource's Condition of type
// ConditionTypeResourceSynced to the supplied status, optional message and
// reason.
//
// TODO(jaypipes): Move to ACK code-gen templates.
func setSyncedCondition(
	r *resource,
	status corev1.ConditionStatus,
	message *string,
	reason *string,
) {
	c := getSyncedCondition(r)
	if c == nil {
		c = &ackv1alpha1.Condition{
			Type: ackv1alpha1.ConditionTypeResourceSynced,
		}
		r.ko.Status.Conditions = append(r.ko.Status.Conditions, c)
	}
	now := metav1.Now()
	c.LastTransitionTime = &now
	c.Status = status
}

// setTerminalCondition sets the resource's Condition of type
// ConditionTypeTerminal to the supplied status, optional message and reason.
//
// TODO(jaypipes): Move to ACK code-gen templates.
func setTerminalCondition(
	r *resource,
	status corev1.ConditionStatus,
	message *string,
	reason *string,
) {
	c := getSyncedCondition(r)
	if c == nil {
		c = &ackv1alpha1.Condition{
			Type: ackv1alpha1.ConditionTypeTerminal,
		}
		r.ko.Status.Conditions = append(r.ko.Status.Conditions, c)
	}
	now := metav1.Now()
	c.LastTransitionTime = &now
	c.Status = status
	c.Message = message
	c.Reason = reason
}
