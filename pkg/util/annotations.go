package util

import (
	"strconv"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

type DeleteInputAnnotationParameters struct {
	SkipFinalSnapshot         *bool
	FinalDBSnapshotIdentifier *string
	DeleteAutomatedBackup     *bool
}

var (
	// If not specified skipFinalSnapshot will be set to true by default.
	//
	// Kept for historical purpuse: This was the value that we set in the generator.yaml
	// before writing this annotation parsing functions.
	//
	//   DeleteDBCluster:
	//     override_values:
	//       # Clearly this is not ideal, but will suffice until we add custom hook
	//       # points to the build_request methods to enable a genmeration of the
	//       # final snapshot identifier to use.
	//       SkipFinalSnapshot: true
	defaultSkipFinalSnapshot = true
)

// parseDeletionAnnotations parses the deletion annotations on the supplied
// resource.
func ParseDeletionAnnotations(annotations map[string]string) (*DeleteInputAnnotationParameters, error) {
	params := &DeleteInputAnnotationParameters{
		SkipFinalSnapshot: &defaultSkipFinalSnapshot,
	}
	if len(annotations) == 0 {
		return params, nil
	}

	// Parse SkipFinalSnapshot annotation
	skipFinalSnapshotAnnotationValue, ok := annotations[svcapitypes.SkipFinalSnapshotAnnotation]
	if ok && skipFinalSnapshotAnnotationValue != "" {
		skipFinalSnapshot, err := strconv.ParseBool(skipFinalSnapshotAnnotationValue)
		if err != nil {
			return nil, err
		}
		params.SkipFinalSnapshot = &skipFinalSnapshot
	}

	// Parse FinalDBSnapshotIdentifier annotation
	finalDBSnapshotIdentifierAnnotationValue, ok := annotations[svcapitypes.FinalDBSnapshotIdentifierAnnotation]
	if ok {
		params.FinalDBSnapshotIdentifier = &finalDBSnapshotIdentifierAnnotationValue
	}

	// Parse DeleteAutomatedBackup annotation
	deleteAutomatedBackupAnnotationValue, ok := annotations[svcapitypes.DeleteAutomatedBackupsAnnotation]
	if ok && deleteAutomatedBackupAnnotationValue != "" {
		deleteAutomatedBackup, err := strconv.ParseBool(deleteAutomatedBackupAnnotationValue)
		if err != nil {
			return nil, err
		}
		params.DeleteAutomatedBackup = &deleteAutomatedBackup
	}
	return params, nil
}
