	if delta.DifferentAt("Spec.EnableCloudwatchLogsExports") {
		cloudwatchLogExportsConfigDesired := desired.ko.Spec.EnableCloudwatchLogsExports
		// Latest log types config
		cloudwatchLogExportsConfigLatest := latest.ko.Spec.EnableCloudwatchLogsExports
		logsTypesToEnable, logsTypesToDisable := getCloudwatchLogExportsConfigDifferences(cloudwatchLogExportsConfigDesired, cloudwatchLogExportsConfigLatest)
		f24 := &svcsdktypes.CloudwatchLogsExportConfiguration{
			EnableLogTypes:  aws.ToStringSlice(logsTypesToEnable),
			DisableLogTypes: aws.ToStringSlice(logsTypesToDisable),
		}
		input.CloudwatchLogsExportConfiguration = f24
	}

	// ModifyDBCluster does not take into account current values when setting these.
	// If one is in the delta need to send all of them even if they have not changed.
	if delta.DifferentAt("Spec.DatabaseInsightsMode") ||
		delta.DifferentAt("Spec.PerformanceInsightsRetentionPeriod") ||
		delta.DifferentAt("Spec.EnablePerformanceInsights") ||
		delta.DifferentAt("Spec.PerformanceInsightsKMSKeyID") {
		if desired.ko.Spec.DatabaseInsightsMode != nil {
			input.DatabaseInsightsMode = svcsdktypes.DatabaseInsightsMode(*desired.ko.Spec.DatabaseInsightsMode)
		}
		if desired.ko.Spec.PerformanceInsightsRetentionPeriod != nil {
			input.PerformanceInsightsRetentionPeriod = aws.Int32(int32(*desired.ko.Spec.PerformanceInsightsRetentionPeriod))
		}
		input.EnablePerformanceInsights = desired.ko.Spec.EnablePerformanceInsights
		input.PerformanceInsightsKMSKeyId = desired.ko.Spec.PerformanceInsightsKMSKeyID
	}
