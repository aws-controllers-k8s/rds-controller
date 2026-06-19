	// CreateOptionGroup does not accept any options; they are configured via
	// ModifyOptionGroup. Rather than make that call here, mark the resource as
	// not yet synced when options are desired so the next reconciliation runs
	// the update path and syncs them through the normal delta flow.
	if len(ko.Spec.Options) > 0 {
		msg := "option group created, options will be configured on the next reconciliation"
		reason := "options pending configuration"
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, &msg, &reason)
	}
