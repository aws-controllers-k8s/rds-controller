	// Do not consider any of the following fields for delta if they are missing in
	// desired(a) but are present in latest(b) because each of these fields is
	// late-initialized
	// This special handling is only needed for DBInstance because late
	// initialized values are not returned after successful ModifyDBInstance
	// call. They are only populated once the DBInstance returns back to
	// available.
	if a.ko.Spec.AvailabilityZone == nil &&
		b.ko.Spec.AvailabilityZone != nil {
		a.ko.Spec.AvailabilityZone = b.ko.Spec.AvailabilityZone
	}
	if a.ko.Spec.BackupTarget == nil &&
		b.ko.Spec.BackupTarget != nil &&
		*b.ko.Spec.BackupTarget == ServiceDefaultBackupTarget {
		a.ko.Spec.BackupTarget = b.ko.Spec.BackupTarget
	}
	if a.ko.Spec.NetworkType == nil &&
		b.ko.Spec.NetworkType != nil &&
		*b.ko.Spec.NetworkType == ServiceDefaultNetworkType {
		a.ko.Spec.NetworkType = b.ko.Spec.NetworkType
	}
	if a.ko.Spec.PerformanceInsightsRetentionPeriod == nil &&
		b.ko.Spec.PerformanceInsightsRetentionPeriod != nil &&
		*b.ko.Spec.PerformanceInsightsRetentionPeriod == ServiceDefaultInsightsRetentionPeriod {
		a.ko.Spec.PerformanceInsightsRetentionPeriod = b.ko.Spec.PerformanceInsightsRetentionPeriod
	}
	if a.ko.Spec.PerformanceInsightsKMSKeyID == nil &&
		b.ko.Spec.PerformanceInsightsKMSKeyID != nil {
		a.ko.Spec.PerformanceInsightsKMSKeyID = b.ko.Spec.PerformanceInsightsKMSKeyID
	}

	// RDS will choose preferred engine minor version if only
	// engine major version is provided and controler should not
	// treat them as different, such as spec has 14, status has 14.1
	// controller should treat them as same
	reconcileEngineVersion(a, b)
    compareTags(delta, a, b)
	compareSecretReferenceChanges(delta, a, b)
