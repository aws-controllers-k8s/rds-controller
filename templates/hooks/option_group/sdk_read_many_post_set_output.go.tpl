	if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
        resourceARN := (*string)(ko.Status.ACKResourceMetadata.ARN)
        tags, err := rm.getTags(ctx, *resourceARN)
        if err != nil {
            return nil, err
        }
        ko.Spec.Tags = tags
	}
	// Project the observed options (the richer DescribeOptionGroups read
	// shape) back into the desired Spec.Options shape so the custom
	// comparison can detect drift between desired and observed options.
	ko.Spec.Options = optionConfigurationsFromObserved(ko.Status.ObservedOptions)
