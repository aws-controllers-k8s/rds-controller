	r := &resource{ko}
	// We expect the DB cluster snapshot to be in 'creating' status since we just
	// issued the call to create it, but I suppose it doesn't hurt to check here.
	if clusterSnapshotCreating(r) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(r, corev1.ConditionFalse, nil, nil)
		return r, nil
	}