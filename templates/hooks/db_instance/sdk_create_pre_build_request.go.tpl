    // if request has DBSnapshotIdentifier spec, create request will call RestoreDBInstanceFromDBSnapshotWithContext
    // instead of normal create api
    created, err = rm.restoreDbInstanceFromDbSnapshot(ctx, desired)
    if created != nil || err != nil {
	    return created, err
    }