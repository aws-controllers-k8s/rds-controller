    // When ModifyDBInstance API is successful, it asynchronously
	// updates the DBInstanceStatus. Requeue to find the current
	// DBInstance status and set Synced condition accordingly
	if err == nil {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
	}