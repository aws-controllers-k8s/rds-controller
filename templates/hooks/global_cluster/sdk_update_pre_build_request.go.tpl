	updatedko := desired.ko.DeepCopy()
	updatedko.Status = latest.ko.Status
	if delta.DifferentAt("Spec.Tags") {
		if err = rm.syncTags(ctx, desired, latest); err != nil {
			return &resource{updatedko}, err
		}
	}
	if !delta.DifferentExcept("Spec.Tags") {
		return &resource{updatedko}, nil
	}