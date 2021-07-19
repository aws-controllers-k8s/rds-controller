	if instanceDeleting(r) {
		return r, requeueWaitWhileDeleting
	}
