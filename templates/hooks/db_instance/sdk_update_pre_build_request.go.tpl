	if instanceDeleting(latest) {
		msg := "DB instance is currently being deleted"
		setSyncedCondition(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitWhileDeleting
	}
	if instanceCreating(latest) {
		msg := "DB instance is currently being created"
		setSyncedCondition(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitUntilCanModify(latest)
	}
	if instanceHasTerminalStatus(latest) {
		msg := "DB instance is in '"+*latest.ko.Status.DBInstanceStatus+"' status"
		setTerminalCondition(desired, corev1.ConditionTrue, &msg, nil)
		setSyncedCondition(desired, corev1.ConditionTrue, nil, nil)
		return desired, nil
	}
