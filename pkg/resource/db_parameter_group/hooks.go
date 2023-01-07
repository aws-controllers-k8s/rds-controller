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

package db_parameter_group

import (
	"context"
	"fmt"
	"sync"
	"time"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go/aws"
	svcsdk "github.com/aws/aws-sdk-go/service/rds"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/rds-controller/pkg/util"
)

const (
	applyTypeStatic        = "static"
	sourceUser             = "user"
	maxResetParametersSize = 20
)

var (
	errUnknownParameter         = fmt.Errorf("unknown parameter")
	errUnmodifiableParameter    = fmt.Errorf("parameter is not modifiable")
	errParamterGroupJustCreated = fmt.Errorf("parameter group just got created")

	requeueWaitWhileCreating = ackrequeue.NeededAfter(
		errParamterGroupJustCreated,
		100*time.Millisecond,
	)
)

func newErrUnknownParameter(name string) error {
	// This is a terminal error because unless the user removes this parameter
	// from their list of parameter overrides, we will not be able to get the
	// resource into a synced state.
	return ackerr.NewTerminalError(
		fmt.Errorf("%w: %s", errUnknownParameter, name),
	)
}

func newErrUnmodifiableParameter(name string) error {
	// This is a terminal error because unless the user removes this parameter
	// from their list of parameter overrides, we will not be able to get the
	// resource into a synced state.
	return ackerr.NewTerminalError(
		fmt.Errorf("%w: %s", errUnmodifiableParameter, name),
	)
}

// customUpdate is required to fix
// https://github.com/aws-controllers-k8s/community/issues/869.
//
// We will need to update parameters in a parameter group using custom logic.
// Until then, however, let's support updating tags for the parameter group.
func (rm *resourceManager) customUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdate")
	defer func() {
		exit(err)
	}()
	if delta.DifferentAt("Spec.Tags") {
		if err = rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.ParameterOverrides") {
		if err = rm.syncParameters(ctx, desired, latest); err != nil {
			return nil, err
		}
	}
	return desired, nil
}

// syncTags keeps the resource's tags in sync
//
// NOTE(jaypipes): RDS' Tagging APIs differ from other AWS APIs in the
// following ways:
//
//  1. The names of the tagging API operations are different. Other APIs use the
//     Tagris `ListTagsForResource`, `TagResource` and `UntagResource` API
//     calls. RDS uses `ListTagsForResource`, `AddTagsToResource` and
//     `RemoveTagsFromResource`.
//
//  2. Even though the name of the `ListTagsForResource` API call is the same,
//     the structure of the input and the output are different from other APIs.
//     For the input, instead of a `ResourceArn` field, RDS names the field
//     `ResourceName`, but actually expects an ARN, not the parameter group
//     name.  This is the same for the `AddTagsToResource` and
//     `RemoveTagsFromResource` input shapes. For the output shape, the field is
//     called `TagList` instead of `Tags` but is otherwise the same struct with
//     a `Key` and `Value` member field.
func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncTags")
	defer func() { exit(err) }()

	arn := (*string)(latest.ko.Status.ACKResourceMetadata.ARN)

	toAdd, toDelete := util.ComputeTagsDelta(
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)

	if len(toDelete) > 0 {
		rlog.Debug("removing tags from parameter group", "tags", toDelete)
		_, err = rm.sdkapi.RemoveTagsFromResourceWithContext(
			ctx,
			&svcsdk.RemoveTagsFromResourceInput{
				ResourceName: arn,
				TagKeys:      toDelete,
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "RemoveTagsFromResource", err)
		if err != nil {
			return err
		}
	}

	// NOTE(jaypipes): According to the RDS API documentation, adding a tag
	// with a new value overwrites any existing tag with the same key. So, we
	// don't need to do anything to "update" a Tag. Simply including it in the
	// AddTagsToResource call is enough.
	if len(toAdd) > 0 {
		rlog.Debug("adding tags to parameter group", "tags", toAdd)
		_, err = rm.sdkapi.AddTagsToResourceWithContext(
			ctx,
			&svcsdk.AddTagsToResourceInput{
				ResourceName: arn,
				Tags:         sdkTagsFromResourceTags(toAdd),
			},
		)
		rm.metrics.RecordAPICall("UPDATE", "AddTagsToResource", err)
		if err != nil {
			return err
		}
	}
	return nil
}

// getTags retrieves the resource's associated tags
func (rm *resourceManager) getTags(
	ctx context.Context,
	resourceARN string,
) ([]*svcapitypes.Tag, error) {
	resp, err := rm.sdkapi.ListTagsForResourceWithContext(
		ctx,
		&svcsdk.ListTagsForResourceInput{
			ResourceName: &resourceARN,
		},
	)
	rm.metrics.RecordAPICall("GET", "ListTagsForResource", err)
	if err != nil {
		return nil, err
	}
	tags := make([]*svcapitypes.Tag, 0, len(resp.TagList))
	for _, tag := range resp.TagList {
		tags = append(tags, &svcapitypes.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return tags, nil
}

// compareTags adds a difference to the delta if the supplied resources have
// different tag collections
func compareTags(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if len(a.ko.Spec.Tags) != len(b.ko.Spec.Tags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	} else if len(a.ko.Spec.Tags) > 0 {
		if !util.EqualTags(a.ko.Spec.Tags, b.ko.Spec.Tags) {
			delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
		}
	}
}

// sdkTagsFromResourceTags transforms a *svcapitypes.Tag array to a *svcsdk.Tag
// array.
func sdkTagsFromResourceTags(
	rTags []*svcapitypes.Tag,
) []*svcsdk.Tag {
	tags := make([]*svcsdk.Tag, len(rTags))
	for i := range rTags {
		tags[i] = &svcsdk.Tag{
			Key:   rTags[i].Key,
			Value: rTags[i].Value,
		}
	}
	return tags
}

func equalStrings(a, b *string) bool {
	if a == nil {
		return b == nil || *b == ""
	}
	return (*a == "" && b == nil) || *a == *b
}

// syncParameters keeps the resource's parameters in sync
//
// RDS does not have a DeleteParameter or DeleteParameterFromParameterGroup API
// call. Instead, you need to call ResetDBParameterGroup with a list of DB
// Parameters that you want RDS to reset to a default value.
func (rm *resourceManager) syncParameters(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncParameters")
	defer func() { exit(err) }()

	groupName := latest.ko.Spec.Name
	family := latest.ko.Spec.Family

	toModify, toDelete := computeParametersDelta(
		desired.ko.Spec.ParameterOverrides, latest.ko.Spec.ParameterOverrides,
	)

	// NOTE(jaypipes): ResetDBParameterGroup and ModifyDBParameterGroup only
	// accept 20 parameters at a time, which is why we "chunk" both the deleted
	// and modified parameter sets.

	if len(toDelete) > 0 {
		chunks := sliceStringChunks(toDelete, maxResetParametersSize)
		for _, chunk := range chunks {
			err = rm.resetParameters(ctx, family, groupName, chunk)
			if err != nil {
				return err
			}
		}
	}

	if len(toModify) > 0 {
		chunks := mapStringChunks(toModify, maxResetParametersSize)
		for _, chunk := range chunks {
			err = rm.modifyParameters(ctx, family, groupName, chunk)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// getParameters retrieves the parameter group's user-defined parameters
// (overrides) and the "statuses" of those parameter overrides.
func (rm *resourceManager) getParameters(
	ctx context.Context,
	groupName *string,
) (
	params map[string]*string,
	paramStatuses []*svcapitypes.Parameter,
	err error,
) {
	var marker *string
	params = make(map[string]*string)
	for {
		resp, err := rm.sdkapi.DescribeDBParametersWithContext(
			ctx,
			&svcsdk.DescribeDBParametersInput{
				DBParameterGroupName: groupName,
				Source:               aws.String(sourceUser),
				Marker:               marker,
			},
		)
		rm.metrics.RecordAPICall("GET", "DescribeDBParameters", err)
		if err != nil {
			return nil, nil, err
		}
		for _, param := range resp.Parameters {
			params[*param.ParameterName] = param.ParameterValue
			p := svcapitypes.Parameter{
				ParameterName:  param.ParameterName,
				ParameterValue: param.ParameterValue,
				ApplyMethod:    param.ApplyMethod,
				ApplyType:      param.ApplyType,
			}
			paramStatuses = append(paramStatuses, &p)
		}
		marker = resp.Marker
		if marker == nil {
			break
		}
	}
	return params, paramStatuses, nil
}

// resetParameters calls the RDS ResetDBParameterGroup API call with a set of
// no more than 20 parameters to reset.
func (rm *resourceManager) resetParameters(
	ctx context.Context,
	family *string,
	groupName *string,
	toDelete []string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.resetParameters")
	defer func() { exit(err) }()

	var pMeta *paramMeta
	inputParams := []*svcsdk.Parameter{}
	for _, paramName := range toDelete {
		// default to this if something goes wrong looking up parameter
		// defaults
		applyMethod := svcsdk.ApplyMethodImmediate
		pMeta, err = cachedParamMeta.get(
			ctx, *family, paramName, rm.getFamilyParameters,
		)
		if err != nil {
			return err
		}
		if !pMeta.isModifiable {
			return newErrUnmodifiableParameter(paramName)
		}
		if !pMeta.isDynamic {
			applyMethod = svcsdk.ApplyMethodPendingReboot
		}
		p := &svcsdk.Parameter{
			ParameterName: aws.String(paramName),
			// TODO(jaypipes): Look up appropriate apply method for this
			// parameter...
			ApplyMethod: aws.String(applyMethod),
		}
		inputParams = append(inputParams, p)
	}

	rlog.Debug(
		"resetting parameters from parameter group",
		"parameters", toDelete,
	)
	_, err = rm.sdkapi.ResetDBParameterGroupWithContext(
		ctx,
		&svcsdk.ResetDBParameterGroupInput{
			DBParameterGroupName: groupName,
			Parameters:           inputParams,
		},
	)
	rm.metrics.RecordAPICall("UPDATE", "ResetDBParameterGroup", err)
	if err != nil {
		return err
	}
	return nil
}

// modifyParameters calls the RDS ModifyDBParameterGroup API call with a set of
// no more than 20 parameters to modify.
func (rm *resourceManager) modifyParameters(
	ctx context.Context,
	family *string,
	groupName *string,
	toModify map[string]*string,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.modifyParameters")
	defer func() { exit(err) }()

	var pMeta *paramMeta
	inputParams := []*svcsdk.Parameter{}
	for paramName, paramValue := range toModify {
		// default to this if something goes wrong looking up parameter
		// defaults
		applyMethod := svcsdk.ApplyMethodImmediate
		pMeta, err = cachedParamMeta.get(
			ctx, *family, paramName, rm.getFamilyParameters,
		)
		if err != nil {
			return err
		}
		if !pMeta.isModifiable {
			return newErrUnmodifiableParameter(paramName)
		}
		if !pMeta.isDynamic {
			applyMethod = svcsdk.ApplyMethodPendingReboot
		}
		p := &svcsdk.Parameter{
			ParameterName:  aws.String(paramName),
			ParameterValue: paramValue,
			// TODO(jaypipes): Look up appropriate apply method for this
			// parameter...
			ApplyMethod: aws.String(applyMethod),
		}
		inputParams = append(inputParams, p)
	}

	rlog.Debug(
		"modifying parameters from parameter group",
		"parameters", toModify,
	)
	_, err = rm.sdkapi.ModifyDBParameterGroupWithContext(
		ctx,
		&svcsdk.ModifyDBParameterGroupInput{
			DBParameterGroupName: groupName,
			Parameters:           inputParams,
		},
	)
	rm.metrics.RecordAPICall("UPDATE", "ModifyDBParameterGroup", err)
	if err != nil {
		return err
	}
	return nil
}

// getFamilyParameters calls the RDS DescribeEngineDefaultParameters API to
// retrieve the set of parameter information for a DB parameter group family.
func (rm *resourceManager) getFamilyParameters(
	ctx context.Context,
	family string,
) (map[string]paramMeta, error) {
	var marker *string
	familyMeta := map[string]paramMeta{}

	for {
		resp, err := rm.sdkapi.DescribeEngineDefaultParametersWithContext(
			ctx,
			&svcsdk.DescribeEngineDefaultParametersInput{
				DBParameterGroupFamily: aws.String(family),
				Marker:                 marker,
			},
		)
		rm.metrics.RecordAPICall("GET", "DescribeEngineDefaultParameters", err)
		if err != nil {
			return nil, err
		}
		for _, param := range resp.EngineDefaults.Parameters {
			pName := *param.ParameterName
			familyMeta[pName] = paramMeta{
				isModifiable: *param.IsModifiable,
				isDynamic:    *param.ApplyType != applyTypeStatic,
			}
		}
		marker = resp.EngineDefaults.Marker
		if marker == nil {
			break
		}
	}
	return familyMeta, nil
}

// computeParametersDelta compares two Parameter arrays and returns the new
// parameters to add, to update and the parameter identifiers to delete
func computeParametersDelta(
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

// sliceStringChunks splits a supplied slice of string pointers into multiple
// slices of string pointers of a given size.
func sliceStringChunks(
	input []string,
	chunkSize int,
) [][]string {
	var chunks [][]string
	for {
		if len(input) == 0 {
			break
		}

		if len(input) < chunkSize {
			chunkSize = len(input)
		}

		chunks = append(chunks, input[0:chunkSize])
		input = input[chunkSize:]
	}

	return chunks
}

// mapStringChunks splits a supplied map of string pointers into multiple
// slices of maps of string pointers of a given size.
func mapStringChunks(
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

type paramMeta struct {
	isModifiable bool
	isDynamic    bool
}

// metaFetcher is the functor we pass to the paramMetaCache that allows it to
// fetch engine default parameter information
type metaFetcher func(ctx context.Context, family string) (map[string]paramMeta, error)

// paramMetaCache stores information about a parameter for a DB parameter group
// family. We use this cached information to determine whether a parameter is
// statically or dynamically defined (whether changes can be applied
// immediately or pending a reboot) and whether a parameter is modifiable.
//
// Keeping things super simple for now and not adding any TTL or expiration
// behaviour to the cache. Engine defaults are pretty static information...
type paramMetaCache struct {
	sync.RWMutex
	hits   uint64
	misses uint64
	cache  map[string]map[string]paramMeta
}

// get retrieves the metadata for a named parameter group family and parameter
// name.
func (c *paramMetaCache) get(
	ctx context.Context,
	family string,
	name string,
	fetcher metaFetcher,
) (*paramMeta, error) {
	var err error
	var found bool
	var metas map[string]paramMeta
	var meta paramMeta

	// We need to release the lock right after the read operation, because
	// loadFamilly will might call a writeLock at L619
	c.RLock()
	metas, found = c.cache[family]
	c.RUnlock()

	if !found {
		c.misses++
		metas, err = c.loadFamily(ctx, family, fetcher)
		if err != nil {
			return nil, err
		}
	}
	meta, found = metas[name]
	if !found {
		return nil, newErrUnknownParameter(name)
	}
	c.hits++
	return &meta, nil
}

// loadFamily fetches parameter information from the AWS RDS
// DescribeEngineDefaultParameters API and caches that information.
func (c *paramMetaCache) loadFamily(
	ctx context.Context,
	family string,
	fetcher metaFetcher,
) (map[string]paramMeta, error) {
	familyMeta, err := fetcher(ctx, family)
	if err != nil {
		return nil, err
	}
	c.Lock()
	defer c.Unlock()
	c.cache[family] = familyMeta
	return familyMeta, nil
}

var cachedParamMeta = paramMetaCache{
	cache: map[string]map[string]paramMeta{},
}
