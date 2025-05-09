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

// Code generated by ack-generate. DO NOT EDIT.

package db_subnet_group

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/rds"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	smithy "github.com/aws/smithy-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/rds-controller/apis/v1alpha1"
)

// Hack to avoid import errors during build...
var (
	_ = &metav1.Time{}
	_ = strings.ToLower("")
	_ = &svcsdk.Client{}
	_ = &svcapitypes.DBSubnetGroup{}
	_ = ackv1alpha1.AWSAccountID("")
	_ = &ackerr.NotFound
	_ = &ackcondition.NotManagedMessage
	_ = &reflect.Value{}
	_ = fmt.Sprintf("")
	_ = &ackrequeue.NoRequeue{}
	_ = &aws.Config{}
)

// sdkFind returns SDK-specific information about a supplied resource
func (rm *resourceManager) sdkFind(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkFind")
	defer func() {
		exit(err)
	}()
	// If any required fields in the input shape are missing, AWS resource is
	// not created yet. Return NotFound here to indicate to callers that the
	// resource isn't yet created.
	if rm.requiredFieldsMissingFromReadManyInput(r) {
		return nil, ackerr.NotFound
	}

	input, err := rm.newListRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DescribeDBSubnetGroupsOutput
	resp, err = rm.sdkapi.DescribeDBSubnetGroups(ctx, input)
	rm.metrics.RecordAPICall("READ_MANY", "DescribeDBSubnetGroups", err)
	if err != nil {
		var awsErr smithy.APIError
		if errors.As(err, &awsErr) && awsErr.ErrorCode() == "DBSubnetGroupNotFoundFault" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := r.ko.DeepCopy()

	found := false
	for _, elem := range resp.DBSubnetGroups {
		if elem.DBSubnetGroupArn != nil {
			if ko.Status.ACKResourceMetadata == nil {
				ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
			}
			tmpARN := ackv1alpha1.AWSResourceName(*elem.DBSubnetGroupArn)
			ko.Status.ACKResourceMetadata.ARN = &tmpARN
		}
		if elem.DBSubnetGroupDescription != nil {
			ko.Spec.Description = elem.DBSubnetGroupDescription
		} else {
			ko.Spec.Description = nil
		}
		if elem.DBSubnetGroupName != nil {
			ko.Spec.Name = elem.DBSubnetGroupName
		} else {
			ko.Spec.Name = nil
		}
		if elem.SubnetGroupStatus != nil {
			ko.Status.SubnetGroupStatus = elem.SubnetGroupStatus
		} else {
			ko.Status.SubnetGroupStatus = nil
		}
		if elem.Subnets != nil {
			f4 := []*svcapitypes.Subnet{}
			for _, f4iter := range elem.Subnets {
				f4elem := &svcapitypes.Subnet{}
				if f4iter.SubnetAvailabilityZone != nil {
					f4elemf0 := &svcapitypes.AvailabilityZone{}
					if f4iter.SubnetAvailabilityZone.Name != nil {
						f4elemf0.Name = f4iter.SubnetAvailabilityZone.Name
					}
					f4elem.SubnetAvailabilityZone = f4elemf0
				}
				if f4iter.SubnetIdentifier != nil {
					f4elem.SubnetIdentifier = f4iter.SubnetIdentifier
				}
				if f4iter.SubnetOutpost != nil {
					f4elemf2 := &svcapitypes.Outpost{}
					if f4iter.SubnetOutpost.Arn != nil {
						f4elemf2.ARN = f4iter.SubnetOutpost.Arn
					}
					f4elem.SubnetOutpost = f4elemf2
				}
				if f4iter.SubnetStatus != nil {
					f4elem.SubnetStatus = f4iter.SubnetStatus
				}
				f4 = append(f4, f4elem)
			}
			ko.Status.Subnets = f4
		} else {
			ko.Status.Subnets = nil
		}
		if elem.SupportedNetworkTypes != nil {
			ko.Status.SupportedNetworkTypes = aws.StringSlice(elem.SupportedNetworkTypes)
		} else {
			ko.Status.SupportedNetworkTypes = nil
		}
		if elem.VpcId != nil {
			ko.Status.VPCID = elem.VpcId
		} else {
			ko.Status.VPCID = nil
		}
		found = true
		break
	}
	if !found {
		return nil, ackerr.NotFound
	}

	rm.setStatusDefaults(ko)
	if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
		resourceARN := (*string)(ko.Status.ACKResourceMetadata.ARN)
		tags, err := rm.getTags(ctx, *resourceARN)
		if err != nil {
			return nil, err
		}
		ko.Spec.Tags = tags
	}

	if ko.Status.Subnets != nil {
		f0 := []*string{}
		for _, subnetIdIter := range ko.Status.Subnets {
			if subnetIdIter.SubnetIdentifier != nil {
				f0 = append(f0, subnetIdIter.SubnetIdentifier)
			}
		}
		ko.Spec.SubnetIDs = f0
	}
	return &resource{ko}, nil
}

// requiredFieldsMissingFromReadManyInput returns true if there are any fields
// for the ReadMany Input shape that are required but not present in the
// resource's Spec or Status
func (rm *resourceManager) requiredFieldsMissingFromReadManyInput(
	r *resource,
) bool {
	return r.ko.Spec.Name == nil

}

// newListRequestPayload returns SDK-specific struct for the HTTP request
// payload of the List API call for the resource
func (rm *resourceManager) newListRequestPayload(
	r *resource,
) (*svcsdk.DescribeDBSubnetGroupsInput, error) {
	res := &svcsdk.DescribeDBSubnetGroupsInput{}

	if r.ko.Spec.Name != nil {
		res.DBSubnetGroupName = r.ko.Spec.Name
	}

	return res, nil
}

// sdkCreate creates the supplied resource in the backend AWS service API and
// returns a copy of the resource with resource fields (in both Spec and
// Status) filled in with values from the CREATE API operation's Output shape.
func (rm *resourceManager) sdkCreate(
	ctx context.Context,
	desired *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkCreate")
	defer func() {
		exit(err)
	}()
	input, err := rm.newCreateRequestPayload(ctx, desired)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.CreateDBSubnetGroupOutput
	_ = resp
	resp, err = rm.sdkapi.CreateDBSubnetGroup(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateDBSubnetGroup", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.DBSubnetGroup.DBSubnetGroupArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.DBSubnetGroup.DBSubnetGroupArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.DBSubnetGroup.DBSubnetGroupDescription != nil {
		ko.Spec.Description = resp.DBSubnetGroup.DBSubnetGroupDescription
	} else {
		ko.Spec.Description = nil
	}
	if resp.DBSubnetGroup.DBSubnetGroupName != nil {
		ko.Spec.Name = resp.DBSubnetGroup.DBSubnetGroupName
	} else {
		ko.Spec.Name = nil
	}
	if resp.DBSubnetGroup.SubnetGroupStatus != nil {
		ko.Status.SubnetGroupStatus = resp.DBSubnetGroup.SubnetGroupStatus
	} else {
		ko.Status.SubnetGroupStatus = nil
	}
	if resp.DBSubnetGroup.Subnets != nil {
		f4 := []*svcapitypes.Subnet{}
		for _, f4iter := range resp.DBSubnetGroup.Subnets {
			f4elem := &svcapitypes.Subnet{}
			if f4iter.SubnetAvailabilityZone != nil {
				f4elemf0 := &svcapitypes.AvailabilityZone{}
				if f4iter.SubnetAvailabilityZone.Name != nil {
					f4elemf0.Name = f4iter.SubnetAvailabilityZone.Name
				}
				f4elem.SubnetAvailabilityZone = f4elemf0
			}
			if f4iter.SubnetIdentifier != nil {
				f4elem.SubnetIdentifier = f4iter.SubnetIdentifier
			}
			if f4iter.SubnetOutpost != nil {
				f4elemf2 := &svcapitypes.Outpost{}
				if f4iter.SubnetOutpost.Arn != nil {
					f4elemf2.ARN = f4iter.SubnetOutpost.Arn
				}
				f4elem.SubnetOutpost = f4elemf2
			}
			if f4iter.SubnetStatus != nil {
				f4elem.SubnetStatus = f4iter.SubnetStatus
			}
			f4 = append(f4, f4elem)
		}
		ko.Status.Subnets = f4
	} else {
		ko.Status.Subnets = nil
	}
	if resp.DBSubnetGroup.SupportedNetworkTypes != nil {
		ko.Status.SupportedNetworkTypes = aws.StringSlice(resp.DBSubnetGroup.SupportedNetworkTypes)
	} else {
		ko.Status.SupportedNetworkTypes = nil
	}
	if resp.DBSubnetGroup.VpcId != nil {
		ko.Status.VPCID = resp.DBSubnetGroup.VpcId
	} else {
		ko.Status.VPCID = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.CreateDBSubnetGroupInput, error) {
	res := &svcsdk.CreateDBSubnetGroupInput{}

	if r.ko.Spec.Description != nil {
		res.DBSubnetGroupDescription = r.ko.Spec.Description
	}
	if r.ko.Spec.Name != nil {
		res.DBSubnetGroupName = r.ko.Spec.Name
	}
	if r.ko.Spec.SubnetIDs != nil {
		res.SubnetIds = aws.ToStringSlice(r.ko.Spec.SubnetIDs)
	}
	if r.ko.Spec.Tags != nil {
		f3 := []svcsdktypes.Tag{}
		for _, f3iter := range r.ko.Spec.Tags {
			f3elem := &svcsdktypes.Tag{}
			if f3iter.Key != nil {
				f3elem.Key = f3iter.Key
			}
			if f3iter.Value != nil {
				f3elem.Value = f3iter.Value
			}
			f3 = append(f3, *f3elem)
		}
		res.Tags = f3
	}

	return res, nil
}

// sdkUpdate patches the supplied resource in the backend AWS service API and
// returns a new resource with updated fields.
func (rm *resourceManager) sdkUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkUpdate")
	defer func() {
		exit(err)
	}()
	input, err := rm.newUpdateRequestPayload(ctx, desired, delta)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.ModifyDBSubnetGroupOutput
	_ = resp
	resp, err = rm.sdkapi.ModifyDBSubnetGroup(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "ModifyDBSubnetGroup", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()
	if delta.DifferentAt("Spec.Tags") {
		if err = rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}

	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.DBSubnetGroup.DBSubnetGroupArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.DBSubnetGroup.DBSubnetGroupArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.DBSubnetGroup.DBSubnetGroupDescription != nil {
		ko.Spec.Description = resp.DBSubnetGroup.DBSubnetGroupDescription
	} else {
		ko.Spec.Description = nil
	}
	if resp.DBSubnetGroup.DBSubnetGroupName != nil {
		ko.Spec.Name = resp.DBSubnetGroup.DBSubnetGroupName
	} else {
		ko.Spec.Name = nil
	}
	if resp.DBSubnetGroup.SubnetGroupStatus != nil {
		ko.Status.SubnetGroupStatus = resp.DBSubnetGroup.SubnetGroupStatus
	} else {
		ko.Status.SubnetGroupStatus = nil
	}
	if resp.DBSubnetGroup.Subnets != nil {
		f4 := []*svcapitypes.Subnet{}
		for _, f4iter := range resp.DBSubnetGroup.Subnets {
			f4elem := &svcapitypes.Subnet{}
			if f4iter.SubnetAvailabilityZone != nil {
				f4elemf0 := &svcapitypes.AvailabilityZone{}
				if f4iter.SubnetAvailabilityZone.Name != nil {
					f4elemf0.Name = f4iter.SubnetAvailabilityZone.Name
				}
				f4elem.SubnetAvailabilityZone = f4elemf0
			}
			if f4iter.SubnetIdentifier != nil {
				f4elem.SubnetIdentifier = f4iter.SubnetIdentifier
			}
			if f4iter.SubnetOutpost != nil {
				f4elemf2 := &svcapitypes.Outpost{}
				if f4iter.SubnetOutpost.Arn != nil {
					f4elemf2.ARN = f4iter.SubnetOutpost.Arn
				}
				f4elem.SubnetOutpost = f4elemf2
			}
			if f4iter.SubnetStatus != nil {
				f4elem.SubnetStatus = f4iter.SubnetStatus
			}
			f4 = append(f4, f4elem)
		}
		ko.Status.Subnets = f4
	} else {
		ko.Status.Subnets = nil
	}
	if resp.DBSubnetGroup.SupportedNetworkTypes != nil {
		ko.Status.SupportedNetworkTypes = aws.StringSlice(resp.DBSubnetGroup.SupportedNetworkTypes)
	} else {
		ko.Status.SupportedNetworkTypes = nil
	}
	if resp.DBSubnetGroup.VpcId != nil {
		ko.Status.VPCID = resp.DBSubnetGroup.VpcId
	} else {
		ko.Status.VPCID = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newUpdateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Update API call for the resource
func (rm *resourceManager) newUpdateRequestPayload(
	ctx context.Context,
	r *resource,
	delta *ackcompare.Delta,
) (*svcsdk.ModifyDBSubnetGroupInput, error) {
	res := &svcsdk.ModifyDBSubnetGroupInput{}

	if r.ko.Spec.Description != nil {
		res.DBSubnetGroupDescription = r.ko.Spec.Description
	}
	if r.ko.Spec.Name != nil {
		res.DBSubnetGroupName = r.ko.Spec.Name
	}
	if r.ko.Spec.SubnetIDs != nil {
		res.SubnetIds = aws.ToStringSlice(r.ko.Spec.SubnetIDs)
	}

	return res, nil
}

// sdkDelete deletes the supplied resource in the backend AWS service API
func (rm *resourceManager) sdkDelete(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkDelete")
	defer func() {
		exit(err)
	}()
	input, err := rm.newDeleteRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DeleteDBSubnetGroupOutput
	_ = resp
	resp, err = rm.sdkapi.DeleteDBSubnetGroup(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteDBSubnetGroup", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteDBSubnetGroupInput, error) {
	res := &svcsdk.DeleteDBSubnetGroupInput{}

	if r.ko.Spec.Name != nil {
		res.DBSubnetGroupName = r.ko.Spec.Name
	}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.DBSubnetGroup,
) {
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if ko.Status.ACKResourceMetadata.Region == nil {
		ko.Status.ACKResourceMetadata.Region = &rm.awsRegion
	}
	if ko.Status.ACKResourceMetadata.OwnerAccountID == nil {
		ko.Status.ACKResourceMetadata.OwnerAccountID = &rm.awsAccountID
	}
	if ko.Status.Conditions == nil {
		ko.Status.Conditions = []*ackv1alpha1.Condition{}
	}
}

// updateConditions returns updated resource, true; if conditions were updated
// else it returns nil, false
func (rm *resourceManager) updateConditions(
	r *resource,
	onSuccess bool,
	err error,
) (*resource, bool) {
	ko := r.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	// Terminal condition
	var terminalCondition *ackv1alpha1.Condition = nil
	var recoverableCondition *ackv1alpha1.Condition = nil
	var syncCondition *ackv1alpha1.Condition = nil
	for _, condition := range ko.Status.Conditions {
		if condition.Type == ackv1alpha1.ConditionTypeTerminal {
			terminalCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeRecoverable {
			recoverableCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeResourceSynced {
			syncCondition = condition
		}
	}
	var termError *ackerr.TerminalError
	if rm.terminalAWSError(err) || err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
		if terminalCondition == nil {
			terminalCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeTerminal,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, terminalCondition)
		}
		var errorMessage = ""
		if err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
			errorMessage = err.Error()
		} else {
			awsErr, _ := ackerr.AWSError(err)
			errorMessage = awsErr.Error()
		}
		terminalCondition.Status = corev1.ConditionTrue
		terminalCondition.Message = &errorMessage
	} else {
		// Clear the terminal condition if no longer present
		if terminalCondition != nil {
			terminalCondition.Status = corev1.ConditionFalse
			terminalCondition.Message = nil
		}
		// Handling Recoverable Conditions
		if err != nil {
			if recoverableCondition == nil {
				// Add a new Condition containing a non-terminal error
				recoverableCondition = &ackv1alpha1.Condition{
					Type: ackv1alpha1.ConditionTypeRecoverable,
				}
				ko.Status.Conditions = append(ko.Status.Conditions, recoverableCondition)
			}
			recoverableCondition.Status = corev1.ConditionTrue
			awsErr, _ := ackerr.AWSError(err)
			errorMessage := err.Error()
			if awsErr != nil {
				errorMessage = awsErr.Error()
			}
			recoverableCondition.Message = &errorMessage
		} else if recoverableCondition != nil {
			recoverableCondition.Status = corev1.ConditionFalse
			recoverableCondition.Message = nil
		}
	}
	// Required to avoid the "declared but not used" error in the default case
	_ = syncCondition
	if terminalCondition != nil || recoverableCondition != nil || syncCondition != nil {
		return &resource{ko}, true // updated
	}
	return nil, false // not updated
}

// terminalAWSError returns awserr, true; if the supplied error is an aws Error type
// and if the exception indicates that it is a Terminal exception
// 'Terminal' exception are specified in generator configuration
func (rm *resourceManager) terminalAWSError(err error) bool {
	if err == nil {
		return false
	}

	var terminalErr smithy.APIError
	if !errors.As(err, &terminalErr) {
		return false
	}
	switch terminalErr.ErrorCode() {
	case "DBSubnetGroupDoesNotCoverEnoughAZs",
		"InvalidSubnet",
		"InvalidParameter",
		"SubnetAlreadyInUse":
		return true
	default:
		return false
	}
}
