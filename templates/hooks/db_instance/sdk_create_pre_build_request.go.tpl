    // if request has DBSnapshotIdentifier spec, create request will call RestoreDBInstanceFromDBSnapshotWithContext
    // instead of normal create api
    if desired.ko.Spec.DBSnapshotIdentifier != nil {
    	created, err = rm.restoreDbInstanceFromDbSnapshot(ctx, desired)
        return created, err
    }
    // if request has SourceDBInstanceIdentifier spec, create request will call CreateDBInstanceReadReplicaWithContext
    // instead of normal create api
    if desired.ko.Spec.SourceDBInstanceIdentifier != nil {
        created, err = rm.createDBInstanceReadReplica(ctx, desired)
        return created, err
    }
