	// We expect the DB instance to be in 'creating' status since we just
	// issued the call to create it, but I suppose it doesn't hurt to check
	// here.
	if instanceCreating(&resource{ko}) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		setSyncedCondition(&resource{ko}, corev1.ConditionFalse, nil, nil)
		return &resource{ko}, nil
	}
