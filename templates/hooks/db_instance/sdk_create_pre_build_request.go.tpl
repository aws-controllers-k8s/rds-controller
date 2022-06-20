    // if request has DBSnapshotIdentifier spec, create request will call RestoreDBInstanceFromDBSnapshotWithContext
    // instead of normal create api
    if desired.ko.Spec.DBSnapshotIdentifier != nil {
        return rm.restoreDbInstanceFromDbSnapshot(ctx, desired)
    }
    // if request has SourceDBInstanceIdentifier spec, create request will call CreateDBInstanceReadReplicaWithContext
    // instead of normal create api
    if desired.ko.Spec.SourceDBInstanceIdentifier != nil {
        return rm.createDBInstanceReadReplica(ctx, desired)
    }
