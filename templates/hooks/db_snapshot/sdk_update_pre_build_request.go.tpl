	if delta.DifferentAt("Spec.Tags") {
		if err = rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}
	if !snapshotAvailable(latest) {
		msg := "DB instance cannot be modifed while in '" + *latest.ko.Status.Status + "' status"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitUntilCanModify(latest)
	}
	if !delta.DifferentExcept("Spec.Tags") {
		// Tags are synced above; ModifyDBSnapshot has nothing to send.
		return desired, nil
	}
