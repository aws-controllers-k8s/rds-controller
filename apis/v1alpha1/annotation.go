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

package v1alpha1

import "fmt"

var (
	// LastAppliedConfigMapAnnotation is the annotation key used to store the namespaced name
	// of the last used secret for setting the master user password of a DBInstance or DBCluster.
	//
	// The secret namespaced name stored in this annotation is used to compute the "reference" delta
	// when the user updates the DBInstance or DBCluster resource.
	//
	// This annotation is only applied by the rds-controller, and should not be modified by the user.
	// In case the user modifies this annotation, the rds-controller may not be able to correctly
	// compute the "reference" delta, and can result in the rds-controller making unnecessary password
	// updates to the DBInstance or DBCluster.
	LastAppliedSecretAnnotation = fmt.Sprintf("%s/last-applied-secret-reference", GroupVersion.Group)
)
