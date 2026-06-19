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

package option_group

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

	"github.com/aws-controllers-k8s/rds-controller/pkg/util"
)

// applyImmediately controls whether ModifyOptionGroup applies option changes
// immediately rather than during the next maintenance window. ACK reconciles
// toward the desired state, so changes are applied immediately.
var applyImmediately = true

// ModifyOptionGroup does not return the updated options, and customUpdate
// returns desired without a subsequent ReadOne. Requeue after syncing options
// so the next reconciliation re-reads the group and refreshes
// Status.ObservedOptions with the server's view.
var (
	errOptionsModified          = fmt.Errorf("options modified, requeuing to refresh status")
	requeueWaitAfterOptionsSync = ackrequeue.NeededAfter(
		errOptionsModified,
		5*time.Second,
	)
)

// syncTags keeps the resource's tags in sync
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
		rlog.Debug("removing tags from option group", "tags", toDelete)
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
		rlog.Debug("adding tags to option group", "tags", toAdd)
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

// customPreCompare runs the resource-specific comparisons that the generated
// delta cannot handle: tags and options are compared with custom logic and so
// are marked compare.is_ignored in generator.yaml.
func customPreCompare(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	compareTags(delta, a, b)
	compareOptions(delta, a, b)
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

// customUpdate handles the OptionGroup update. The generated ModifyOptionGroup
// payload only carries the option group name, so tags and options are synced
// via dedicated API calls here. Description, EngineName and MajorEngineVersion
// are immutable (enforced by CRD validation) and cannot be modified.
func (rm *resourceManager) customUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdate")
	defer func() { exit(err) }()

	if delta.DifferentAt("Spec.Tags") {
		if err = rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.Options") {
		if err = rm.syncOptions(ctx, desired, latest); err != nil {
			return nil, err
		}
		// ModifyOptionGroup does not return the updated options, so requeue to
		// re-read the group and refresh Status.ObservedOptions.
		return desired, requeueWaitAfterOptionsSync
	}
	return desired, nil
}

// syncOptions keeps the option group's options in sync by computing the set of
// options to include (added or reconfigured) and the set of options to remove,
// then issuing a single ModifyOptionGroup call. CreateOptionGroup cannot set
// options, so newly-created groups are requeued (synced=false) and their
// options are configured here on the subsequent update reconciliation.
func (rm *resourceManager) syncOptions(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncOptions")
	defer func() { exit(err) }()

	var latestOptions []*svcapitypes.OptionConfiguration
	if latest != nil {
		latestOptions = latest.ko.Spec.Options
	}

	toInclude := sdkOptionConfigurationsFromResource(desired.ko.Spec.Options)
	toRemove := optionsToRemove(desired.ko.Spec.Options, latestOptions)

	if len(toInclude) == 0 && len(toRemove) == 0 {
		return nil
	}

	rlog.Debug(
		"modifying option group options",
		"to_include", len(toInclude),
		"to_remove", toRemove,
	)
	_, err = rm.sdkapi.ModifyOptionGroup(
		ctx,
		&svcsdk.ModifyOptionGroupInput{
			OptionGroupName:  desired.ko.Spec.Name,
			ApplyImmediately: &applyImmediately,
			OptionsToInclude: toInclude,
			OptionsToRemove:  toRemove,
		},
	)
	rm.metrics.RecordAPICall("UPDATE", "ModifyOptionGroup", err)
	return err
}

// optionsToRemove returns the names of options present in latest but no longer
// present in desired. These need to be removed from the option group.
func optionsToRemove(
	desired []*svcapitypes.OptionConfiguration,
	latest []*svcapitypes.OptionConfiguration,
) []string {
	desiredNames := map[string]bool{}
	for _, o := range desired {
		if o != nil && o.OptionName != nil {
			desiredNames[*o.OptionName] = true
		}
	}
	toRemove := []string{}
	for _, o := range latest {
		if o == nil || o.OptionName == nil {
			continue
		}
		if !desiredNames[*o.OptionName] {
			toRemove = append(toRemove, *o.OptionName)
		}
	}
	return toRemove
}

// compareOptions adds a difference to the delta when the desired options differ
// from the options observed on the option group. The observed options (the
// richer DescribeOptionGroups read shape) carry server-applied defaults that
// the user never specified, so the comparison is asymmetric: a difference is
// flagged only when a desired option (or one of its specified settings) is
// missing from, or differs in, the observed options, or when an observed option
// is no longer desired.
//
// a is the desired resource, b is the latest (observed) resource, matching the
// argument order of newResourceDelta.
func compareOptions(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	desired := a.ko.Spec.Options
	latest := b.ko.Spec.Options

	if len(desired) == 0 && len(latest) == 0 {
		return
	}

	latestByName := map[string]*svcapitypes.OptionConfiguration{}
	for _, o := range latest {
		if o != nil && o.OptionName != nil {
			latestByName[*o.OptionName] = o
		}
	}

	// Every desired option must be present in the observed options, and the
	// settings the user specified must match.
	desiredNames := map[string]bool{}
	for _, d := range desired {
		if d == nil || d.OptionName == nil {
			continue
		}
		desiredNames[*d.OptionName] = true
		l, ok := latestByName[*d.OptionName]
		if !ok || !desiredOptionMatches(d, l) {
			delta.Add("Spec.Options", desired, latest)
			return
		}
	}

	// Any observed option that is no longer desired is drift.
	for _, l := range latest {
		if l == nil || l.OptionName == nil {
			continue
		}
		if !desiredNames[*l.OptionName] {
			delta.Add("Spec.Options", desired, latest)
			return
		}
	}
}

// desiredOptionMatches reports whether the observed option satisfies the
// desired option. Only the fields the user specified on the desired option are
// compared; unspecified fields (and server-applied default settings) are
// ignored so that defaults do not cause perpetual drift.
func desiredOptionMatches(
	desired *svcapitypes.OptionConfiguration,
	latest *svcapitypes.OptionConfiguration,
) bool {
	if desired == nil || latest == nil {
		return false
	}
	if desired.OptionVersion != nil {
		if latest.OptionVersion == nil || *desired.OptionVersion != *latest.OptionVersion {
			return false
		}
	}
	if desired.Port != nil {
		if latest.Port == nil || *desired.Port != *latest.Port {
			return false
		}
	}
	if len(desired.DBSecurityGroupMemberships) > 0 &&
		!equalStringSets(desired.DBSecurityGroupMemberships, latest.DBSecurityGroupMemberships) {
		return false
	}
	if len(desired.VPCSecurityGroupMemberships) > 0 &&
		!equalStringSets(desired.VPCSecurityGroupMemberships, latest.VPCSecurityGroupMemberships) {
		return false
	}
	return desiredSettingsMatch(desired.OptionSettings, latest.OptionSettings)
}

// desiredSettingsMatch reports whether every desired option setting is present
// with the same value in the observed settings. Observed settings the user did
// not specify (server defaults) are ignored.
func desiredSettingsMatch(
	desired []*svcapitypes.OptionSetting,
	latest []*svcapitypes.OptionSetting,
) bool {
	if len(desired) == 0 {
		return true
	}
	latestByName := map[string]*svcapitypes.OptionSetting{}
	for _, s := range latest {
		if s != nil && s.Name != nil {
			latestByName[*s.Name] = s
		}
	}
	for _, d := range desired {
		if d == nil || d.Name == nil {
			continue
		}
		l, ok := latestByName[*d.Name]
		if !ok {
			return false
		}
		if !ackcompare.HasNilDifference(d.Value, l.Value) {
			if d.Value != nil && l.Value != nil && *d.Value != *l.Value {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

// equalStringSets reports whether two slices contain the same set of string
// values, ignoring order and nil entries.
func equalStringSets(a, b []*string) bool {
	as := map[string]bool{}
	for _, s := range a {
		if s != nil {
			as[*s] = true
		}
	}
	bs := map[string]bool{}
	for _, s := range b {
		if s != nil {
			bs[*s] = true
		}
	}
	if len(as) != len(bs) {
		return false
	}
	for k := range as {
		if !bs[k] {
			return false
		}
	}
	return true
}

// optionConfigurationsFromObserved projects the observed options (the richer
// DescribeOptionGroups read shape) into the desired Spec.Options shape so the
// custom comparison can diff desired against observed.
func optionConfigurationsFromObserved(
	observed []*svcapitypes.Option,
) []*svcapitypes.OptionConfiguration {
	if observed == nil {
		return nil
	}
	out := make([]*svcapitypes.OptionConfiguration, 0, len(observed))
	for _, o := range observed {
		if o == nil {
			continue
		}
		cfg := &svcapitypes.OptionConfiguration{
			OptionName:    o.OptionName,
			OptionVersion: o.OptionVersion,
			Port:          o.Port,
		}
		for _, m := range o.DBSecurityGroupMemberships {
			if m != nil {
				cfg.DBSecurityGroupMemberships = append(
					cfg.DBSecurityGroupMemberships, m.DBSecurityGroupName,
				)
			}
		}
		for _, m := range o.VPCSecurityGroupMemberships {
			if m != nil {
				cfg.VPCSecurityGroupMemberships = append(
					cfg.VPCSecurityGroupMemberships, m.VPCSecurityGroupID,
				)
			}
		}
		cfg.OptionSettings = o.OptionSettings
		out = append(out, cfg)
	}
	return out
}

// sdkOptionConfigurationsFromResource transforms the desired Spec.Options into
// the SDK OptionConfiguration shape used by ModifyOptionGroup.OptionsToInclude.
func sdkOptionConfigurationsFromResource(
	options []*svcapitypes.OptionConfiguration,
) []svcsdktypes.OptionConfiguration {
	if len(options) == 0 {
		return nil
	}
	out := make([]svcsdktypes.OptionConfiguration, 0, len(options))
	for _, o := range options {
		if o == nil {
			continue
		}
		cfg := svcsdktypes.OptionConfiguration{
			OptionName:    o.OptionName,
			OptionVersion: o.OptionVersion,
		}
		if o.Port != nil {
			port := int32(*o.Port)
			cfg.Port = &port
		}
		for _, m := range o.DBSecurityGroupMemberships {
			if m != nil {
				cfg.DBSecurityGroupMemberships = append(cfg.DBSecurityGroupMemberships, *m)
			}
		}
		for _, m := range o.VPCSecurityGroupMemberships {
			if m != nil {
				cfg.VpcSecurityGroupMemberships = append(cfg.VpcSecurityGroupMemberships, *m)
			}
		}
		for _, s := range o.OptionSettings {
			if s == nil {
				continue
			}
			cfg.OptionSettings = append(cfg.OptionSettings, svcsdktypes.OptionSetting{
				Name:  s.Name,
				Value: s.Value,
			})
		}
		out = append(out, cfg)
	}
	return out
}
