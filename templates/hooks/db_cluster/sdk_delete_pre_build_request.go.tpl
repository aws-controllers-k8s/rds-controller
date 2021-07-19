	if clusterDeleting(r) {
		return r, requeueWaitWhileDeleting
	}
