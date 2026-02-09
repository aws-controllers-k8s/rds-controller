	if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
		resourceARN := (*string)(ko.Status.ACKResourceMetadata.ARN)
		tags, err := rm.getTags(ctx, *resourceARN)
		if err != nil {
			return nil, err
		}
		ko.Spec.Tags = tags
	}
	if ko.Spec.Name != nil {
		groupName := ko.Spec.Name
		family := ko.Spec.Family
		desiredParams := ko.Spec.ParameterOverrides
		params, paramStatuses, err := rm.getParameters(
			ctx, groupName, family, desiredParams,
		)
		if err != nil {
			return nil, err
		}
		ko.Spec.ParameterOverrides = params
		ko.Status.ParameterOverrideStatuses = paramStatuses
	}
