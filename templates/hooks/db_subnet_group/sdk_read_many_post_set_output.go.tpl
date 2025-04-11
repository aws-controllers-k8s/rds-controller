	if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
        resourceARN := (*string)(ko.Status.ACKResourceMetadata.ARN)
        tags, err := rm.getTags(ctx, *resourceARN)
        if err != nil {
            return nil, err
        }
        ko.Spec.Tags = tags
	}

	if ko.Status.Subnets != nil {
		f0 := []*string{}
		for _, subnetIdIter := range ko.Status.Subnets {
			if subnetIdIter.SubnetIdentifier != nil {
				f0 = append(f0, subnetIdIter.SubnetIdentifier)
			}
		}
		ko.Spec.SubnetIDs = f0
	}