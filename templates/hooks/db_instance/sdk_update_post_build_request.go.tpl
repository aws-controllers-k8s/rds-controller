	// ModifyDBInstance call will return ValidationError when the
	// ModifyDBInstanceRequest contains the same DBSubnetGroupName
	// as the DBInstance. So, if there is no delta between
	// desired and latest for Spec.DBSubnetGroupName, exclude it
	// from ModifyDBInstanceRequest
	if !delta.DifferentAt("Spec.DBSubnetGroupName") {
		input.DBSubnetGroupName = nil
	}
