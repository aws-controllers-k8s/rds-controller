	if clusterHasTerminalStatus(latest) {
		msg := "DB cluster is in '" + *latest.ko.Status.Status + "' status"
		ackcondition.SetTerminal(desired, corev1.ConditionTrue, &msg, nil)
		ackcondition.SetSynced(desired, corev1.ConditionTrue, nil, nil)
		return desired, nil
	}
	if delta.DifferentAt("Spec.Tags") {
		if err = rm.syncTags(ctx, desired, latest); err != nil {
			return nil, err
		}
	}
	if !delta.DifferentExcept("Spec.Tags") {
		// Tags are synced above; ModifyDBCluster has nothing to send.
		return desired, nil
	}
