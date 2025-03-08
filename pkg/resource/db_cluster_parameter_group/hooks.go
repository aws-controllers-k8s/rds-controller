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

package db_cluster_parameter_group

import (
	"context"
	"fmt"
	"time"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/rds"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws-controllers-k8s/rds-controller/pkg/util"
)

const (
	applyTypeStatic        = "static"
	sourceUser             = "user"
	maxResetParametersSize = 20
)

var (
	// cache of parameter defaults
	cachedParamMeta = util.ParamMetaCache{
		Cache: map[string]map[string]util.ParamMeta{},
	}

	errParameterGroupJustCreated = fmt.Errorf("parameter group just got created")
	requeueWaitWhileCreating     = ackrequeue.NeededAfter(
		errParameterGroupJustCreated,
		100*time.Millisecond,
	)
)

// customUpdate is required to fix
// https://github.com/aws-controllers-k8s/community/issues/869.
//
// We will need to update parameters in a cluster parameter group using custom logic.
// Until then, however, let's support updating tags for the cluster parameter group.
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
		rlog.Debug("removing tags from cluster parameter group", "tags", toDelete)
		_, err = rm.sdkapi.RemoveTagsFromResource(
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
		rlog.Debug("adding tags to cluster parameter group", "tags", toAdd)
		_, err = rm.sdkapi.AddTagsToResource(
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
	resp, err := rm.sdkapi.ListTagsForResource(
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
) []svcsdktypes.Tag {
	tags := make([]svcsdktypes.Tag, len(rTags))
	for i := range rTags {
		tags[i] = svcsdktypes.Tag{
			Key:   rTags[i].Key,
			Value: rTags[i].Value,
		}
	}
	return tags
}

// syncParameters keeps the resource's parameters in sync
//
// RDS does not have a DeleteParameter or DeleteParameterFromParameterGroup API
// call. Instead, you need to call ResetDBClusterParameterGroup with a list of
// DB Cluster Parameters that you want RDS to reset to a default value.
func (rm *resourceManager) syncParameters(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncParameters")
	defer func() { exit(err) }()

	groupName := desired.ko.Spec.Name
	family := desired.ko.Spec.Family

	// If there are no parameter overrides in the desired state,
	// consider this a valid state and return success
	if len(desired.ko.Spec.ParameterOverrides) == 0 {
		return nil
	}

	desiredOverrides := desired.ko.Spec.ParameterOverrides
	latestOverrides := util.Parameters{}
	// In the create code paths, we pass a nil latest...
	if latest != nil {
		latestOverrides = latest.ko.Spec.ParameterOverrides
	}

	toModify, _, toDelete := util.GetParametersDifference(
		desiredOverrides, latestOverrides,
	)

	if len(toDelete) > 0 {
		err = rm.resetParameters(ctx, family, groupName, toDelete)
		if err != nil {
			return err
		}
	}

	if len(toModify) > 0 {
		err = rm.modifyParameters(ctx, family, groupName, toModify)
		if err != nil {
			return err
		}
	}
	return nil
}

// getParameters retrieves the cluster parameter group's user-defined parameters
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
		resp, err := rm.sdkapi.DescribeDBClusterParameters(
			ctx,
			&svcsdk.DescribeDBClusterParametersInput{
				DBClusterParameterGroupName: groupName,
				Source:                      aws.String(sourceUser),
				Marker:                      marker,
			},
		)
		rm.metrics.RecordAPICall("GET", "DescribeDBClusterParameters", err)
		if err != nil {
			return nil, nil, err
		}
		for _, param := range resp.Parameters {
			params[*param.ParameterName] = param.ParameterValue
			p := svcapitypes.Parameter{
				ParameterName:  param.ParameterName,
				ParameterValue: param.ParameterValue,
				ApplyMethod:    aws.String(string(param.ApplyMethod)),
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

// resetParameters calls the RDS ResetDBClusterParameterGroup API call
func (rm *resourceManager) resetParameters(
	ctx context.Context,
	family *string,
	groupName *string,
	toDelete util.Parameters,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.resetParameters")
	defer func() { exit(err) }()

	var pMeta *util.ParamMeta
	inputParams := []svcsdktypes.Parameter{}
	for paramName, _ := range toDelete {
		// default to this if something goes wrong looking up parameter
		// defaults
		applyMethod := svcsdktypes.ApplyMethodImmediate
		pMeta, err = cachedParamMeta.Get(
			ctx, *family, paramName, rm.getFamilyParameters,
		)
		if err != nil {
			return err
		}
		if !pMeta.IsModifiable {
			return util.NewErrUnmodifiableParameter(paramName)
		}
		if !pMeta.IsDynamic {
			applyMethod = svcsdktypes.ApplyMethodPendingReboot
		}
		p := svcsdktypes.Parameter{
			ParameterName: aws.String(paramName),
			ApplyMethod:   svcsdktypes.ApplyMethod(applyMethod),
		}
		inputParams = append(inputParams, p)
	}

	rlog.Debug(
		"resetting parameters from cluster parameter group",
		"parameters", toDelete,
	)
	_, err = rm.sdkapi.ResetDBClusterParameterGroup(
		ctx,
		&svcsdk.ResetDBClusterParameterGroupInput{
			DBClusterParameterGroupName: groupName,
			Parameters:                  inputParams,
		},
	)
	rm.metrics.RecordAPICall("UPDATE", "ResetDBClusterParameterGroup", err)
	if err != nil {
		return err
	}
	return nil
}

// modifyParameters calls the RDS ModifyDBClusterParameterGroup API call
func (rm *resourceManager) modifyParameters(
	ctx context.Context,
	family *string,
	groupName *string,
	toModify util.Parameters,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.modifyParameters")
	defer func() { exit(err) }()

	var pMeta *util.ParamMeta
	inputParams := []svcsdktypes.Parameter{}
	for paramName, paramValue := range toModify {
		// default to "immediate" if something goes wrong looking up defaults
		applyMethod := svcsdktypes.ApplyMethodImmediate
		pMeta, err = cachedParamMeta.Get(
			ctx, *family, paramName, rm.getFamilyParameters,
		)
		if err != nil {
			return err
		}
		if !pMeta.IsModifiable {
			return util.NewErrUnmodifiableParameter(paramName)
		}
		if !pMeta.IsDynamic {
			applyMethod = svcsdktypes.ApplyMethodPendingReboot
		}
		p := svcsdktypes.Parameter{
			ParameterName:  aws.String(paramName),
			ParameterValue: paramValue,
			ApplyMethod:    svcsdktypes.ApplyMethod(applyMethod),
		}
		inputParams = append(inputParams, p)
	}

	rlog.Debug(
		"modifying parameters from parameter group",
		"parameters", toModify,
	)
	_, err = rm.sdkapi.ModifyDBClusterParameterGroup(
		ctx,
		&svcsdk.ModifyDBClusterParameterGroupInput{
			DBClusterParameterGroupName: groupName,
			Parameters:                  inputParams,
		},
	)
	rm.metrics.RecordAPICall("UPDATE", "ModifyDBClusterParameterGroup", err)
	if err != nil {
		return err
	}
	return nil
}

// getFamilyParameters calls the RDS DescribeEngineDefaultClusterParameters API to
// retrieve the set of parameter information for a DB cluster parameter group family.
func (rm *resourceManager) getFamilyParameters(
	ctx context.Context,
	family string,
) (map[string]util.ParamMeta, error) {
	var marker *string
	familyMeta := map[string]util.ParamMeta{}

	for {
		resp, err := rm.sdkapi.DescribeEngineDefaultClusterParameters(
			ctx,
			&svcsdk.DescribeEngineDefaultClusterParametersInput{
				DBParameterGroupFamily: aws.String(family),
				Marker:                 marker,
			},
		)
		rm.metrics.RecordAPICall("GET", "DescribeEngineDefaultClusterParameters", err)
		if err != nil {
			return nil, err
		}
		for _, param := range resp.EngineDefaults.Parameters {
			pName := *param.ParameterName
			familyMeta[pName] = util.ParamMeta{
				IsModifiable: *param.IsModifiable,
				IsDynamic:    *param.ApplyType != applyTypeStatic,
			}
		}
		marker = resp.EngineDefaults.Marker
		if marker == nil {
			break
		}
	}
	return familyMeta, nil
}
