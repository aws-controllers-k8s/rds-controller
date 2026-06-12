    compareTags(delta, a, b)

    // Handle special case for StorageType field for Aurora engines
    // When StorageType is set to "aurora" (default), the API doesn't return it
    if isAuroraEngine(b.ko.Spec.Engine) && (a.ko.Spec.StorageType != nil && *a.ko.Spec.StorageType == "aurora" && b.ko.Spec.StorageType == nil) {
            b.ko.Spec.StorageType = aws.String("aurora")
    }

    // Clear AllocatedStorage on the desired side for Aurora; the observed side
    // is already cleared on create/read (v1.9.0+). This mainly covers clusters
    // created before that fix, which retain a stale allocatedStorage in spec
    // that otherwise produces a delta every reconcile ModifyDBCluster can't
    // resolve, leaving the cluster perpetually out of sync.
    clearAuroraAllocatedStorage(a.ko)

    // When autoMinorVersionUpgrade is enabled and the engine version
    // difference is only a minor version change (same major version),
    // normalize the desired engine version to match the latest. This
    // prevents the delta from firing on every reconcile when AWS
    // auto-upgrades the minor version.
    if a.ko.Spec.EngineVersion != nil && b.ko.Spec.EngineVersion != nil {
        autoMinorVersionUpgrade := true
        if a.ko.Spec.AutoMinorVersionUpgrade != nil {
            autoMinorVersionUpgrade = *a.ko.Spec.AutoMinorVersionUpgrade
        }
        if !requireEngineVersionUpdate(a.ko.Spec.EngineVersion, b.ko.Spec.EngineVersion, autoMinorVersionUpgrade) {
            a.ko.Spec.EngineVersion = b.ko.Spec.EngineVersion
        }
    }

    compareSecretReferenceChanges(delta, a, b)
