	// ModifyDBInstance call will return ValidationError when the
	// ModifyDBInstanceRequest contains the same DBSubnetGroupName
	// as the DBInstance. So, if there is no delta between
	// desired and latest for Spec.DBSubnetGroupName, exclude it
	// from ModifyDBInstanceRequest
	if !delta.DifferentAt("Spec.DBSubnetGroupName") {
		input.DBSubnetGroupName = nil
	}

        // RDS will not compare diff value and accept any modify db call
        // for below values, MonitoringInterval, CACertificateIdentifier
        // and user master password, NetworkType
        // hence if there is no delta between desired 
        // and latest, exclude it from ModifyDBInstanceRequest
        if !delta.DifferentAt("Spec.MonitoringInterval") {
		input.MonitoringInterval = nil
	}
        if !delta.DifferentAt("Spec.CACertificateIdentifier") {
		input.CACertificateIdentifier = nil
	}
        if !delta.DifferentAt("Spec.MasterUserPassword.Name") {
		input.MasterUserPassword = nil
	}
        if !delta.DifferentAt("Spec.NetworkType") {
                input.NetworkType = nil
        }
