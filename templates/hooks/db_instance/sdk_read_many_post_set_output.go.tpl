	// DescribeDBInstances returns an array of DBInstance structs that contains
	// the *previously set* values for various mutable fields. This is
	// problematic because it causes a "flopping" behaviour when the user has
	// modified a Spec field from value A to value B but the output shape from
	// ModifyDBInstance for that field contains value A, the standard SetOutput
	// Go code generated above will set the Spec field to the *old* value
	// again. The next time the reconciler runs, it will read the latest
	// observed resource, see a difference between the desired and the latest
	// state (that actually does not exist because the difference is comparing
	// the value of the fields before they were changed) and attempt to modify
	// the field from value B to value A again, causing a flop loop.
	//
	// Luckily, the Output shape's DBInstance struct contains a
	// `PendingModifiedValues` struct which contains those field values that
	// the user specified. So, we can use these to "reset" the Spec back to the
	// appropriate user-specified values.
	pmv := ko.Status.PendingModifiedValues
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
        // NOTE(jaypipes): Handle when aws-sdk-go update
		//if pmv.Iops != nil {
		//	ko.Spec.IOPS = pmv.Iops
		//}
		if pmv.LicenseModel != nil {
			ko.Spec.LicenseModel = pmv.LicenseModel
		}
		if pmv.MasterUserPassword != nil {
			// NOTE(jaypipes): Type mismatch with Spec and
			// PendingModifiedValues, so just reset to the original...
			ko.Spec.MasterUserPassword = r.ko.Spec.MasterUserPassword
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
	if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
		resourceARN := (*string)(ko.Status.ACKResourceMetadata.ARN)
		tags, err := rm.getTags(ctx, *resourceARN)
		if err != nil {
			return nil, err
		}
		ko.Spec.Tags = tags
	}
	if !instanceAvailable(&resource{ko}) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
	}
	if len(r.ko.Spec.VPCSecurityGroupIDs) > 0 {
		// If the desired resource has security groups specified then update the spec of the latest resource with the
		// security groups from the status. This is done so that when an instance is created without security groups
		// and gets a default security group attached to it, it is not overwritten with no security groups from the
		// desired resource.
		sgIDs := make([]*string, len(ko.Status.VPCSecurityGroups))
		for i, sg := range ko.Status.VPCSecurityGroups {
			id := *sg.VPCSecurityGroupID
			sgIDs[i] = &id
		}
		ko.Spec.VPCSecurityGroupIDs = sgIDs
	}
	if r.ko.Spec.DBParameterGroupName != nil {
		// If the desired resource has db parameter group name specified then update the spec of the latest 
		// resource with the value from the status.
		if len(ko.Status.DBParameterGroups) > 0 {
			ko.Spec.DBParameterGroupName = ko.Status.DBParameterGroups[0].DBParameterGroupName
		}
	}
  
  // We currently do not set spec value for EnableCloudwatchLogsExports
  // and instead only set the status field.
  // Adding DBInstance.enableCloudwatchLogsExports doesn't update the RDS instance
  // https://github.com/aws-controllers-k8s/community/issues/2128
  ko.Spec.EnableCloudwatchLogsExports = ko.Status.EnabledCloudwatchLogsExports
