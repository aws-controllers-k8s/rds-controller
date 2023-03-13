	if proxyDeleting(r) {
		return r, requeueWaitWhileDeleting
	}
