    compareTags(delta, a, b)

    // Handle special case for StorageType field for Aurora engines
    // When StorageType is set to "aurora" (default), the API doesn't return it

    isAuroraEngine := (b.ko.Spec.Engine != nil && (*b.ko.Spec.Engine == "aurora-mysql" || *b.ko.Spec.Engine == "aurora-postgresql"))

    if isAuroraEngine && (a.ko.Spec.StorageType != nil && *a.ko.Spec.StorageType == "aurora" && b.ko.Spec.StorageType == nil) {
            b.ko.Spec.StorageType = aws.String("aurora")
    }

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
