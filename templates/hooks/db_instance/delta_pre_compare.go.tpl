    // Do not consider AvailabilityZone field for delta if it is missing in
    // desired(a) and is present in latest(b) because AvailabilityZone field is
    // late-initialized
    // This special handling is only needed for DBInstance because a requeue
    // error needs to be returned even after successful ModifyDBInstance call.
    // See sdk_update_post_set_output.go.tpl for more details.
    // The requeue error from update prevents the late initialization in
    // reconciler.go and without ignoring AvailabilityZone for delta , the
    // reconciler will keep updating the dbinstance and constantly requeueing it.
    if a != nil && a.ko.Spec.AvailabilityZone == nil && b != nil && b.ko.Spec.AvailabilityZone != nil {
        a.ko.Spec.AvailabilityZone = b.ko.Spec.AvailabilityZone
    }
    compareTags(delta, a, b)
