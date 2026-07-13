	if delta.DifferentAt("Spec.EnableCloudwatchLogsExports") {
		cloudwatchLogExportsConfigDesired := desired.ko.Spec.EnableCloudwatchLogsExports
		cloudwatchLogExportsConfigLatest := latest.ko.Spec.EnableCloudwatchLogsExports
		logsTypesToEnable, logsTypesToDisable := getCloudwatchLogExportsConfigDifferences(cloudwatchLogExportsConfigDesired, cloudwatchLogExportsConfigLatest)
		f24 := &svcsdktypes.CloudwatchLogsExportConfiguration{
			EnableLogTypes:  aws.ToStringSlice(logsTypesToEnable),
			DisableLogTypes: aws.ToStringSlice(logsTypesToDisable),
		}
		input.CloudwatchLogsExportConfiguration = f24
	}
	// AllowMajorVersionUpgrade is set conditionally in sdk.go newUpdateRequestPayload
	// only when Spec.EngineVersion has changed, to avoid a reconcile loop.
