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

package db_subnet_group

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

const testSubnetGroupARN = "arn:aws:rds:us-east-1:123456789012:subgrp:test-subnet-group"

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

// newTestSubnetGroup returns a subnet group with the supplied tags and an ARN
// set in its status, as syncTags requires.
func newTestSubnetGroup(tags []*svcapitypes.Tag) *resource {
	arn := ackv1alpha1.AWSResourceName(testSubnetGroupARN)
	return &resource{
		ko: &svcapitypes.DBSubnetGroup{
			Spec: svcapitypes.DBSubnetGroupSpec{
				Name:        aws.String("test-subnet-group"),
				Description: aws.String("test"),
				SubnetIDs:   []*string{aws.String("subnet-a"), aws.String("subnet-b")},
				Tags:        tags,
			},
			Status: svcapitypes.DBSubnetGroupStatus{
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
func actionsOf(client *stubHTTPClient) []string {
	actions := make([]string, 0, len(client.requests))
	for _, form := range client.requests {
		actions = append(actions, form.Get("Action"))
	}
	return actions
}

func TestSdkUpdateTagsOnlySkipsModify(t *testing.T) {
	ctx := context.Background()
	client := &stubHTTPClient{}
	rm := newTestResourceManager(client)
	desired := newTestSubnetGroup([]*svcapitypes.Tag{newTestTag("env", "prod")})
	latest := newTestSubnetGroup([]*svcapitypes.Tag{newTestTag("env", "dev")})
	delta := ackcompare.NewDelta()
	delta.Add("Spec.Tags", desired.ko.Spec.Tags, latest.ko.Spec.Tags)

	_, err := rm.sdkUpdate(ctx, desired, latest, delta)
	assert.NoError(t, err)
	assert.Equal(t, []string{"AddTagsToResource"}, actionsOf(client))
}

func TestSdkUpdateTagsAndFieldCallsModify(t *testing.T) {
	ctx := context.Background()
	client := &stubHTTPClient{
		responses: map[string]string{
			"ModifyDBSubnetGroup": `<ModifyDBSubnetGroupResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/"><ModifyDBSubnetGroupResult><DBSubnetGroup><DBSubnetGroupName>test-subnet-group</DBSubnetGroupName><DBSubnetGroupDescription>new</DBSubnetGroupDescription></DBSubnetGroup></ModifyDBSubnetGroupResult><ResponseMetadata><RequestId>test-request-id</RequestId></ResponseMetadata></ModifyDBSubnetGroupResponse>`,
		},
	}
	rm := newTestResourceManager(client)
	desired := newTestSubnetGroup([]*svcapitypes.Tag{newTestTag("env", "prod")})
	desired.ko.Spec.Description = aws.String("new")
	latest := newTestSubnetGroup([]*svcapitypes.Tag{newTestTag("env", "dev")})
	delta := ackcompare.NewDelta()
	delta.Add("Spec.Tags", desired.ko.Spec.Tags, latest.ko.Spec.Tags)
	delta.Add("Spec.Description", desired.ko.Spec.Description, latest.ko.Spec.Description)

	_, err := rm.sdkUpdate(ctx, desired, latest, delta)
	assert.NoError(t, err)
	assert.Equal(t, []string{"AddTagsToResource", "ModifyDBSubnetGroup"}, actionsOf(client))
	assert.Equal(t, "new", client.requests[1].Get("DBSubnetGroupDescription"))
}
