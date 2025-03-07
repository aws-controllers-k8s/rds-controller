    compareTags(delta, a, b)

    // Handle special case for StorageType field for Aurora engines
    // When StorageType is set to "aurora" (default), the API doesn't return it

    isAuroraEngine := (b.ko.Spec.Engine != nil && (*b.ko.Spec.Engine == "aurora-mysql" || *b.ko.Spec.Engine == "aurora-postgresql"))
    
    if isAuroraEngine && (a.ko.Spec.StorageType != nil && *a.ko.Spec.StorageType == "aurora" && b.ko.Spec.StorageType == nil) {
            b.ko.Spec.StorageType = aws.String("aurora")
    }  

    compareSecretReferenceChanges(delta, a, b)
