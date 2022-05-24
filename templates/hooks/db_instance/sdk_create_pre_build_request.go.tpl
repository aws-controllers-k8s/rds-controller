    created, err = rm.restoreDbInstanceFromDbSnapshot(ctx, desired)
    if created != nil || err != nil {
	    return created, err
    }