	if !clusterAvailable(&resource{ko}) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		setSyncedCondition(&resource{ko}, corev1.ConditionFalse, nil, nil)
	} else {
		setSyncedCondition(&resource{ko}, corev1.ConditionTrue, nil, nil)
	}

