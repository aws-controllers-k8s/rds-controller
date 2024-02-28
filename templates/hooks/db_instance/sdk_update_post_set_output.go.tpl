	// ModifyDBInstance returns a DBInstance struct that contains the
	// *previously set* values for various mutable fields. This is problematic
	// because it causes a "flopping" behaviour when the user has modified a
	// Spec field from value A to value B but the output shape from
	// ModifyDBInstance for that field contains value A, the standard SetOutput
	// Go code generated above will set the Spec field to the *old* value
	// again. The next time the reconciler runs, it will attempt to modify the
	// field from value B to value A again, causing a flop loop.
	//
	// Luckily, the Output shape's DBInstance struct contains a
	// `PendingModifiedValues` struct which contains those field values that
	// the user specified. So, we can use these to "reset" the Spec back to the
	// appropriate user-specific values.
	pmv := resp.DBInstance.PendingModifiedValues
	if pmv != nil {
		if pmv.AllocatedStorage != nil {
			ko.Spec.AllocatedStorage = pmv.AllocatedStorage
		}
        // NOTE(jaypipes): Handle when aws-sdk-go update
		//if pmv.AutomationMode != nil {
		//	ko.Spec.AutomationMode = pmv.AutomationMode
		//}
		if pmv.BackupRetentionPeriod != nil {
			ko.Spec.BackupRetentionPeriod = pmv.BackupRetentionPeriod
		}
		if pmv.CACertificateIdentifier != nil {
			ko.Spec.CACertificateIdentifier = pmv.CACertificateIdentifier
		}
		if pmv.DBInstanceClass != nil {
			ko.Spec.DBInstanceClass = pmv.DBInstanceClass
		}
		if pmv.DBInstanceIdentifier != nil {
			ko.Spec.DBInstanceIdentifier = pmv.DBInstanceIdentifier
		}
		if pmv.DBSubnetGroupName != nil {
			ko.Spec.DBSubnetGroupName = pmv.DBSubnetGroupName
		}
		if pmv.EngineVersion != nil {
			ko.Spec.EngineVersion = pmv.EngineVersion
		}
        // NOTE(jaypipes): Handle when aws-sdk-go update
		//if pmv.IAMDatabaseAuthenticationEnabled != nil {
		//	ko.Spec.IAMDatabaseAuthenticationEnabled = pmv.IAMDatabaseAuthenticationEnabled
		//}
		if pmv.Iops != nil {
			ko.Spec.IOPS = pmv.Iops
		}
		if pmv.LicenseModel != nil {
			ko.Spec.LicenseModel = pmv.LicenseModel
		}
		if pmv.MasterUserPassword != nil {
			// NOTE(jaypipes): Type mismatch with Spec and
			// PendingModifiedValues, so just reset to the desired...
			ko.Spec.MasterUserPassword = desired.ko.Spec.MasterUserPassword
		}
		if pmv.MultiAZ != nil {
			ko.Spec.MultiAZ = pmv.MultiAZ
		}
        // NOTE(jaypipes): Handle when aws-sdk-go update
		//if pmv.PendingCloudwatchLogsExports != nil {
		//	ko.Spec.PendingCloudwatchLogsExports = pmv.PendingCloudwatchLogsExports
		//}
		if pmv.Port != nil {
			ko.Spec.Port = pmv.Port
		}
        // NOTE(jaypipes): Handle when aws-sdk-go update
		//if pmv.ProcessorFeatures != nil {
		//	ko.Spec.ProcessorFeatures = pmv.ProcessorFeatures
		//}
        // NOTE(jaypipes): Handle when aws-sdk-go update
		//if pmv.ResumeFullAutomationModeTime != nil {
		//	ko.Spec.ResumeFullAutomationModeTime = pmv.ResumeFullAutomationModeTime
		//}
		if pmv.StorageThroughput != nil {
			ko.Spec.StorageThroughput = pmv.StorageThroughput
		}
		if pmv.StorageType != nil {
			ko.Spec.StorageType = pmv.StorageType
		}
	}
	// When ModifyDBInstance API is successful, it asynchronously
	// updates the DBInstanceStatus. Requeue to find the current
	// DBInstance status and set Synced condition accordingly
	if err == nil {
		// set the last-applied-secret-reference annotation on the DB instance
		// resource.
		r := &resource{ko}
		setLastAppliedSecretReferenceAnnotation(r)
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(r, corev1.ConditionFalse, nil, nil)
	}
