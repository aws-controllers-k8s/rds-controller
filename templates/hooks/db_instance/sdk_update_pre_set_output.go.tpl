	ko.Status = latest.ko.Status
	setLastAppliedSecretReferenceAnnotation(&resource{ko})
	// Setting resource synced condition to false will trigger a requeue of
	// the resource. No need to return a requeue error here.
	ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
	// return &resource{ko}, nil
