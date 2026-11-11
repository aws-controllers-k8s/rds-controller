	res := desired.ko.DeepCopy()
	res.Status = latest.ko.Status

	// DeletionProtection of an Aurora or Multi-AZ DB cluster member is set on
	// its DBCluster, and ModifyDBInstance rejects it for a member. The normal
	// delta already ignores it for members (customPreCompare); the pre-delete
	// delta does not, so skip an update that would only change it.
	if latest.ko.Spec.DBClusterIdentifier != nil &&
		!delta.DifferentExcept("Spec.DeletionProtection") {
		return &resource{res}, nil
	}

	if instanceDeleting(latest) {
		msg := "DB instance is currently being deleted"
		ackcondition.SetSynced(&resource{res}, corev1.ConditionFalse, &msg, nil)
		return &resource{res}, requeueWaitWhileDeleting
	}
	if instanceCreating(latest) {
		msg := "DB instance is currently being created"
		ackcondition.SetSynced(&resource{res}, corev1.ConditionFalse, &msg, nil)
		return &resource{res}, requeueWaitUntilCanModify(latest)
	}
	if instanceHasTerminalStatus(latest) {
		msg := "DB instance is in '"+*latest.ko.Status.DBInstanceStatus+"' status"
		ackcondition.SetTerminal(&resource{res}, corev1.ConditionTrue, &msg, nil)
		ackcondition.SetSynced(&resource{res}, corev1.ConditionTrue, nil, nil)
		return &resource{res}, nil
	}
	if !instanceAvailable(latest) && !needStorageUpdate(latest, delta) {
		msg := "DB instance cannot be modifed while in '" + *latest.ko.Status.DBInstanceStatus + "' status"
		ackcondition.SetSynced(&resource{res}, corev1.ConditionFalse, &msg, nil)
		return &resource{res}, requeueWaitUntilCanModify(latest)
	}
	if delta.DifferentAt("Spec.Tags") {
		if err = rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}
