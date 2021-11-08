	if clusterDeleting(latest) {
		msg := "DB cluster is currently being deleted"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitWhileDeleting
	}
	if clusterCreating(latest) {
		msg := "DB cluster is currently being created"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitUntilCanModify(latest)
	}
	if clusterHasTerminalStatus(latest) {
		msg := "DB cluster is in '"+*latest.ko.Status.Status+"' status"
		ackconditoin.SetTerminal(desired, corev1.ConditionTrue, &msg, nil)
		ackcondition.SetSynced(desired, corev1.ConditionTrue, nil, nil)
		return desired, nil
	}
