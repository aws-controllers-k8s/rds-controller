	if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
        resourceARN := (*string)(ko.Status.ACKResourceMetadata.ARN)
        tags, err := rm.getTags(ctx, *resourceARN)
        if err != nil {
            return nil, err
        }
        ko.Spec.Tags = tags
	}
	if !clusterAvailable(&resource{ko}) {
		// Setting resource synced condition to false will trigger a requeue of
		// the resource. No need to return a requeue error here.
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
	} else {
		ackcondition.SetSynced(&resource{ko}, corev1.ConditionTrue, nil, nil)
	}
	if len(r.ko.Spec.VPCSecurityGroupIDs) > 0 {
		// If the desired resource has security groups specified then update the spec of the latest resource with the
		// value from the status. This is done so that when a cluster is created without security groups and gets a
		// default security group attached to it, it is not overwritten with empty security groups from the
		// desired resource.
		sgIDs := make([]*string, len(ko.Status.VPCSecurityGroups))
		for i, sg := range ko.Status.VPCSecurityGroups {
			id := *sg.VPCSecurityGroupID
			sgIDs[i] = &id
		}
		ko.Spec.VPCSecurityGroupIDs = sgIDs
	}
	if r.ko.Spec.DBClusterParameterGroupName != nil {
		// If the desired resource has db cluster parameter group name specified then update the spec of the latest 
		// resource with the value from the status.
		ko.Spec.DBClusterParameterGroupName = ko.Status.DBClusterParameterGroup
	}

	if r.ko.Spec.EnableIAMDatabaseAuthentication != nil  {
	    // If the desired resource has IAM authentication explicitly enabled or disabled then update the spec of the
	    // latest resource with the value from the status.
		ko.Spec.EnableIAMDatabaseAuthentication = ko.Status.IAMDatabaseAuthenticationEnabled
	}

	ko.Spec.EnableCloudwatchLogsExports = ko.Status.EnabledCloudwatchLogsExports 
