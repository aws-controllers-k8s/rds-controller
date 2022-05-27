    // if request has DBSnapshotIdentifier spec, create request will call RestoreDBInstanceFromDBSnapshotWithContext
    // instead of normal create api
    if desired.ko.Spec.DBSnapshotIdentifier != nil {
    	created, err = rm.restoreDbInstanceFromDbSnapshot(ctx, desired)
        return created, err
    }
