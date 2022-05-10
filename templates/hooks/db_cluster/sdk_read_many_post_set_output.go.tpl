	if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
        resourceARN := (*string)(ko.Status.ACKResourceMetadata.ARN)
        tags, err := rm.getTags(ctx, *resourceARN)
        if err != nil {
            return nil, err
        }
        ko.Spec.Tags = tags
	}
	if !clusterAvailable(&resource{ko}) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
	} else {
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionTrue, nil, nil)
	}

