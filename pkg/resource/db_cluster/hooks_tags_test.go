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

package db_cluster

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackmetrics "github.com/aws-controllers-k8s/runtime/pkg/metrics"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

const testClusterARN = "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster"

// recordingHTTPClient captures the form-encoded body of every RDS Query
// protocol request and replies with a minimal valid XML response for the
// requested action.
type recordingHTTPClient struct {
	requests []url.Values
}

func (c *recordingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}
	c.requests = append(c.requests, form)
	respBody := fmt.Sprintf(
		`<%[1]sResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/"><ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata></%[1]sResponse>`,
		form.Get("Action"),
	)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(respBody)),
		Request:    req,
	}, nil
}

// newTestResourceManager returns a resourceManager whose RDS client sends all
// of its requests to the supplied recordingHTTPClient.
func newTestResourceManager(client *recordingHTTPClient) *resourceManager {
	return &resourceManager{
		metrics: ackmetrics.NewMetrics("rds"),
		sdkapi: svcsdk.NewFromConfig(aws.Config{
			Region:      "us-east-1",
			Credentials: aws.AnonymousCredentials{},
			HTTPClient:  client,
		}),
	}
}

// newTestCluster returns a resource with the supplied tags and an ARN set in
// its status, as syncTags requires.
func newTestCluster(tags []*svcapitypes.Tag) *resource {
	arn := ackv1alpha1.AWSResourceName(testClusterARN)
	return &resource{
		ko: &svcapitypes.DBCluster{
			Spec: svcapitypes.DBClusterSpec{
				DBClusterIdentifier: aws.String("test-cluster"),
				Tags:                tags,
			},
			Status: svcapitypes.DBClusterStatus{
				ACKResourceMetadata: &ackv1alpha1.ResourceMetadata{
					ARN: &arn,
				},
			},
		},
	}
}

func newTestTag(key, value string) *svcapitypes.Tag {
	return &svcapitypes.Tag{Key: aws.String(key), Value: aws.String(value)}
}

// actionsOf returns the RDS API action of each recorded request, in order.
func actionsOf(client *recordingHTTPClient) []string {
	actions := make([]string, 0, len(client.requests))
	for _, form := range client.requests {
		actions = append(actions, form.Get("Action"))
	}
	return actions
}

func TestSyncTags(t *testing.T) {
	ctx := context.Background()

	t.Run("no difference makes no AWS calls", func(t *testing.T) {
		client := &recordingHTTPClient{}
		rm := newTestResourceManager(client)
		desired := newTestCluster([]*svcapitypes.Tag{newTestTag("k1", "v1")})
		latest := newTestCluster([]*svcapitypes.Tag{newTestTag("k1", "v1")})

		err := rm.syncTags(ctx, desired, latest)

		assert.NoError(t, err)
		assert.Empty(t, client.requests)
	})

	t.Run("new tag calls AddTagsToResource", func(t *testing.T) {
		client := &recordingHTTPClient{}
		rm := newTestResourceManager(client)
		desired := newTestCluster([]*svcapitypes.Tag{newTestTag("k1", "v1")})
		latest := newTestCluster(nil)

		err := rm.syncTags(ctx, desired, latest)

		assert.NoError(t, err)
		assert.Equal(t, []string{"AddTagsToResource"}, actionsOf(client))
		form := client.requests[0]
		assert.Equal(t, testClusterARN, form.Get("ResourceName"))
		assert.Equal(t, "k1", form.Get("Tags.Tag.1.Key"))
		assert.Equal(t, "v1", form.Get("Tags.Tag.1.Value"))
	})

	t.Run("removed tag calls RemoveTagsFromResource", func(t *testing.T) {
		client := &recordingHTTPClient{}
		rm := newTestResourceManager(client)
		desired := newTestCluster(nil)
		latest := newTestCluster([]*svcapitypes.Tag{newTestTag("k1", "v1")})

		err := rm.syncTags(ctx, desired, latest)

		assert.NoError(t, err)
		assert.Equal(t, []string{"RemoveTagsFromResource"}, actionsOf(client))
		form := client.requests[0]
		assert.Equal(t, testClusterARN, form.Get("ResourceName"))
		assert.Equal(t, "k1", form.Get("TagKeys.member.1"))
	})

	t.Run("changed value calls AddTagsToResource only", func(t *testing.T) {
		client := &recordingHTTPClient{}
		rm := newTestResourceManager(client)
		desired := newTestCluster([]*svcapitypes.Tag{newTestTag("k1", "v2")})
		latest := newTestCluster([]*svcapitypes.Tag{newTestTag("k1", "v1")})

		err := rm.syncTags(ctx, desired, latest)

		assert.NoError(t, err)
		assert.Equal(t, []string{"AddTagsToResource"}, actionsOf(client))
		form := client.requests[0]
		assert.Equal(t, "k1", form.Get("Tags.Tag.1.Key"))
		assert.Equal(t, "v2", form.Get("Tags.Tag.1.Value"))
	})

	t.Run("desired tag with nil key is ignored", func(t *testing.T) {
		client := &recordingHTTPClient{}
		rm := newTestResourceManager(client)
		desired := newTestCluster([]*svcapitypes.Tag{{Value: aws.String("v1")}})
		latest := newTestCluster(nil)

		assert.NotPanics(t, func() {
			err := rm.syncTags(ctx, desired, latest)
			assert.NoError(t, err)
			assert.Empty(t, client.requests)
		})
	})
}

func TestCompareTags(t *testing.T) {
	tests := []struct {
		name      string
		a         []*svcapitypes.Tag
		b         []*svcapitypes.Tag
		wantDelta bool
	}{
		{
			name: "equal in different order",
			a: []*svcapitypes.Tag{
				newTestTag("k1", "v1"),
				newTestTag("k2", "v2"),
			},
			b: []*svcapitypes.Tag{
				newTestTag("k2", "v2"),
				newTestTag("k1", "v1"),
			},
			wantDelta: false,
		},
		{
			name:      "changed value",
			a:         []*svcapitypes.Tag{newTestTag("k1", "v1")},
			b:         []*svcapitypes.Tag{newTestTag("k1", "v2")},
			wantDelta: true,
		},
		{
			name: "tag added",
			a: []*svcapitypes.Tag{
				newTestTag("k1", "v1"),
				newTestTag("k2", "v2"),
			},
			b:         []*svcapitypes.Tag{newTestTag("k1", "v1")},
			wantDelta: true,
		},
		{
			name: "tag removed",
			a:    []*svcapitypes.Tag{newTestTag("k1", "v1")},
			b: []*svcapitypes.Tag{
				newTestTag("k1", "v1"),
				newTestTag("k2", "v2"),
			},
			wantDelta: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta := ackcompare.NewDelta()

			compareTags(delta, newTestCluster(tt.a), newTestCluster(tt.b))

			assert.Equal(t, tt.wantDelta, delta.DifferentAt("Spec.Tags"))
		})
	}
}
