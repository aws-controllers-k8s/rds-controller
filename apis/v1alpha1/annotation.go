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
	// SkipFinalSnapshot is the annotation key used to skip the final snapshot when deleting a DBInstance
	// or DBCluster. If this annotation is set to "true", the final snapshot will be skipped. The default
	// value is "true" - meaning that when the annotation is not present, the final snapshot will be skipped.
	SkipFinalSnapshotAnnotation = fmt.Sprintf("%s/skip-final-snapshot", GroupVersion.Group)
	// FinalDBSnapshotIdentifier is the annotation key used to specify the final snapshot identifier when
	// deleting a DBInstance or DBCluster. If this annotation is set, the final snapshot will be created with
	// the specified identifier.
	//
	// If the SkipFinalSnapshot annotation is set to "true", this annotation will be ignored.
	FinalDBSnapshotIdentifierAnnotation = fmt.Sprintf("%s/final-db-snapshot-identifier", GroupVersion.Group)
	// DeleteAutomatedBackups is the annotation key used to specify whether automated backups should be
	// deleted when deleting a DBInstance or DBCluster. If this annotation is set to "true", automated backups
	// will be deleted. The default value is "false" - meaning that when the annotation is not present, automated
	// backups will not be deleted.
	DeleteAutomatedBackupsAnnotation = fmt.Sprintf("%s/delete-automated-backups", GroupVersion.Group)
)
