    // if request has SnapshotIdentifier spec, create request will call RestoreDBClusterFromSnapshotWithContext
    // instead of normal create api
    if desired.ko.Spec.SnapshotIdentifier != nil {
        return rm.restoreDbClusterFromSnapshot(ctx, desired)
    }

    // if request has SourceDBClusterIdentifier spec, create request will call RestoreDBClusterToPointInTimeWithContext
    // instead of normal create api
    if desired.ko.Spec.SourceDBClusterIdentifier != nil {
        return rm.restoreDbClusterToPointInTime(ctx, desired)
    }
