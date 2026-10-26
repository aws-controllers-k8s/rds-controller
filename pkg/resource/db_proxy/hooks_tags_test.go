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

package db_proxy

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

const testProxyARN = "arn:aws:rds:us-east-1:123456789012:proxy:test-proxy"

// stubHTTPClient captures the form-encoded body of every RDS Query protocol
// request and replies with the XML body registered for the requested action in
// responses. Actions without a registered body get an empty response.
type stubHTTPClient struct {
	requests  []url.Values
	responses map[string]string
}

func (c *stubHTTPClient) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}
	c.requests = append(c.requests, form)
	action := form.Get("Action")
	respBody, ok := c.responses[action]
	if !ok {
		respBody = fmt.Sprintf(
			`<%[1]sResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/"><ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata></%[1]sResponse>`,
			action,
		)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(respBody)),
		Request:    req,
	}, nil
}

// newTestResourceManager returns a resourceManager whose RDS client sends all
// of its requests to the supplied stubHTTPClient.
func newTestResourceManager(client *stubHTTPClient) *resourceManager {
	return &resourceManager{
		metrics: ackmetrics.NewMetrics("rds"),
		sdkapi: svcsdk.NewFromConfig(aws.Config{
			Region:      "us-east-1",
			Credentials: aws.AnonymousCredentials{},
			HTTPClient:  client,
		}),
	}
}

// newTestProxy returns an available resource with the supplied tags and an ARN
// set in its status, as syncTags requires.
func newTestProxy(tags []*svcapitypes.Tag) *resource {
	arn := ackv1alpha1.AWSResourceName(testProxyARN)
	return &resource{
		ko: &svcapitypes.DBProxy{
			Spec: svcapitypes.DBProxySpec{
				Name: aws.String("test-proxy"),
				Tags: tags,
			},
			Status: svcapitypes.DBProxyStatus{
				ACKResourceMetadata: &ackv1alpha1.ResourceMetadata{
					ARN: &arn,
				},
				Status: aws.String("available"),
			},
		},
	}
}

func newTestTag(key, value string) *svcapitypes.Tag {
	return &svcapitypes.Tag{Key: aws.String(key), Value: aws.String(value)}
}

// actionsOf returns the RDS API action of each recorded request, in order.
func actionsOf(client *stubHTTPClient) []string {
	actions := make([]string, 0, len(client.requests))
	for _, form := range client.requests {
		actions = append(actions, form.Get("Action"))
	}
	return actions
}

func TestSdkFindPopulatesTagsFromAWS(t *testing.T) {
	ctx := context.Background()
	client := &stubHTTPClient{
		responses: map[string]string{
			"DescribeDBProxies":   `<DescribeDBProxiesResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/"><DescribeDBProxiesResult><DBProxies><member><DBProxyName>test-proxy</DBProxyName><DBProxyArn>` + testProxyARN + `</DBProxyArn><Status>available</Status></member></DBProxies></DescribeDBProxiesResult><ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata></DescribeDBProxiesResponse>`,
			"ListTagsForResource": `<ListTagsForResourceResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/"><ListTagsForResourceResult><TagList><Tag><Key>k</Key><Value>aws</Value></Tag></TagList></ListTagsForResourceResult><ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata></ListTagsForResourceResponse>`,
		},
	}
	rm := newTestResourceManager(client)
	desired := newTestProxy([]*svcapitypes.Tag{newTestTag("k", "desired")})

	latest, err := rm.sdkFind(ctx, desired)

	assert.NoError(t, err)
	assert.Equal(t, []*svcapitypes.Tag{newTestTag("k", "aws")}, latest.ko.Spec.Tags)
	assert.Contains(t, actionsOf(client), "ListTagsForResource")
}

func TestSdkUpdateTagsOnlySkipsModify(t *testing.T) {
	ctx := context.Background()
	client := &stubHTTPClient{
		responses: map[string]string{
			"AddTagsToResource":      `<AddTagsToResourceResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/"><ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata></AddTagsToResourceResponse>`,
			"RemoveTagsFromResource": `<RemoveTagsFromResourceResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/"><ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata></RemoveTagsFromResourceResponse>`,
		},
	}
	rm := newTestResourceManager(client)
	desired := newTestProxy([]*svcapitypes.Tag{newTestTag("env", "prod")})
	latest := newTestProxy([]*svcapitypes.Tag{newTestTag("env", "dev")})
	delta := ackcompare.NewDelta()
	delta.Add("Spec.Tags", latest.ko.Spec.Tags, desired.ko.Spec.Tags)

	_, err := rm.sdkUpdate(ctx, desired, latest, delta)

	assert.NoError(t, err)
	actions := actionsOf(client)
	assert.NotContains(t, actions, "ModifyDBProxy")
	assert.Contains(t, actions, "AddTagsToResource")
}
