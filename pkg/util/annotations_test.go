package util_test

import (
	"reflect"
	"testing"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/rds-controller/pkg/util"
	"github.com/aws/aws-sdk-go/aws"
)

func TestParseDeletionAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		want        *util.DeleteInputAnnotationParameters
		wantErr     bool
	}{
		{
			name:        "no annotations",
			annotations: map[string]string{},
			want: &util.DeleteInputAnnotationParameters{
				SkipFinalSnapshot:         aws.Bool(true),
				FinalDBSnapshotIdentifier: nil,
				DeleteAutomatedBackup:     nil,
			},
			wantErr: false,
		},
		{
			name: "all annotations set - turn on all snapshot options",
			annotations: map[string]string{
				svcapitypes.SkipFinalSnapshotAnnotation:         "false",
				svcapitypes.FinalDBSnapshotIdentifierAnnotation: "final-snapshot",
				svcapitypes.DeleteAutomatedBackupsAnnotation:    "false",
			},
			want: &util.DeleteInputAnnotationParameters{
				SkipFinalSnapshot:         aws.Bool(false),
				FinalDBSnapshotIdentifier: aws.String("final-snapshot"),
				DeleteAutomatedBackup:     aws.Bool(false),
			},
			wantErr: false,
		},
		{
			name: "all annotations set - turn off all snapshot options",
			annotations: map[string]string{
				svcapitypes.SkipFinalSnapshotAnnotation:      "true",
				svcapitypes.DeleteAutomatedBackupsAnnotation: "true",
			},
			want: &util.DeleteInputAnnotationParameters{
				SkipFinalSnapshot:         aws.Bool(true),
				FinalDBSnapshotIdentifier: nil,
				DeleteAutomatedBackup:     aws.Bool(true),
			},
			wantErr: false,
		},
		{
			name: "invalid SkipFinalSnapshot annotation",
			annotations: map[string]string{
				svcapitypes.SkipFinalSnapshotAnnotation: "invalid",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid DeleteAutomatedBackups annotation",
			annotations: map[string]string{
				svcapitypes.DeleteAutomatedBackupsAnnotation: "invalid",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := util.ParseDeletionAnnotations(tt.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDeletionAnnotations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDeletionAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}
