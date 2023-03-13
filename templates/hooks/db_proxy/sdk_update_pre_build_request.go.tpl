	if proxyDeleting(latest) {
		msg := "DB proxy is currently being deleted"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitWhileDeleting
	}
	if proxyCreating(latest) {
		msg := "DB proxy is currently being created"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitUntilCanModify(latest)
	}
	if proxyHasTerminalStatus(latest) {
		msg := "DB proxy is in '"+*latest.ko.Status.Status+"' status"
		ackcondition.SetTerminal(desired, corev1.ConditionTrue, &msg, nil)
		ackcondition.SetSynced(desired, corev1.ConditionTrue, nil, nil)
		return desired, nil
	}
	if !proxyAvailable(latest) {
		msg := "DB proxy cannot be modifed while in '" + *latest.ko.Status.Status + "' status"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitUntilCanModify(latest)
	}