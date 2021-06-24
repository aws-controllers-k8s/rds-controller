	if clusterDeleting(r) {
		return requeueWaitWhileDeleting
	}
