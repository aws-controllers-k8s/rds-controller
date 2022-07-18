    // if request has SnapshotIdentifier spec, create request will call RestoreDBClusterFromSnapshotWithContext
    // instead of normal create api
    if desired.ko.Spec.SnapshotIdentifier != nil {
        return rm.restoreDbClusterFromSnapshot(ctx, desired)
    }
