// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package v1alpha1

type ActivityStreamMode string

const (
	ActivityStreamMode_sync  ActivityStreamMode = "sync"
	ActivityStreamMode_async ActivityStreamMode = "async"
)

type ActivityStreamStatus string

const (
	ActivityStreamStatus_stopped  ActivityStreamStatus = "stopped"
	ActivityStreamStatus_starting ActivityStreamStatus = "starting"
	ActivityStreamStatus_started  ActivityStreamStatus = "started"
	ActivityStreamStatus_stopping ActivityStreamStatus = "stopping"
)

type ApplyMethod string

const (
	ApplyMethod_immediate      ApplyMethod = "immediate"
	ApplyMethod_pending_reboot ApplyMethod = "pending-reboot"
)

type AuthScheme string

const (
	AuthScheme_SECRETS AuthScheme = "SECRETS"
)

type AutomationMode string

const (
	AutomationMode_full       AutomationMode = "full"
	AutomationMode_all_paused AutomationMode = "all-paused"
)

type CustomEngineVersionStatus string

const (
	CustomEngineVersionStatus_available               CustomEngineVersionStatus = "available"
	CustomEngineVersionStatus_inactive                CustomEngineVersionStatus = "inactive"
	CustomEngineVersionStatus_inactive_except_restore CustomEngineVersionStatus = "inactive-except-restore"
)

type DBProxyEndpointStatus string

const (
	DBProxyEndpointStatus_available                    DBProxyEndpointStatus = "available"
	DBProxyEndpointStatus_modifying                    DBProxyEndpointStatus = "modifying"
	DBProxyEndpointStatus_incompatible_network         DBProxyEndpointStatus = "incompatible-network"
	DBProxyEndpointStatus_insufficient_resource_limits DBProxyEndpointStatus = "insufficient-resource-limits"
	DBProxyEndpointStatus_creating                     DBProxyEndpointStatus = "creating"
	DBProxyEndpointStatus_deleting                     DBProxyEndpointStatus = "deleting"
)

type DBProxyEndpointTargetRole string

const (
	DBProxyEndpointTargetRole_READ_WRITE DBProxyEndpointTargetRole = "READ_WRITE"
	DBProxyEndpointTargetRole_READ_ONLY  DBProxyEndpointTargetRole = "READ_ONLY"
)

type DBProxyStatus string

const (
	DBProxyStatus_available                    DBProxyStatus = "available"
	DBProxyStatus_modifying                    DBProxyStatus = "modifying"
	DBProxyStatus_incompatible_network         DBProxyStatus = "incompatible-network"
	DBProxyStatus_insufficient_resource_limits DBProxyStatus = "insufficient-resource-limits"
	DBProxyStatus_creating                     DBProxyStatus = "creating"
	DBProxyStatus_deleting                     DBProxyStatus = "deleting"
	DBProxyStatus_suspended                    DBProxyStatus = "suspended"
	DBProxyStatus_suspending                   DBProxyStatus = "suspending"
	DBProxyStatus_reactivating                 DBProxyStatus = "reactivating"
)

type EngineFamily string

const (
	EngineFamily_MYSQL      EngineFamily = "MYSQL"
	EngineFamily_POSTGRESQL EngineFamily = "POSTGRESQL"
)

type FailoverStatus string

const (
	FailoverStatus_pending      FailoverStatus = "pending"
	FailoverStatus_failing_over FailoverStatus = "failing-over"
	FailoverStatus_cancelling   FailoverStatus = "cancelling"
)

type IAMAuthMode string

const (
	IAMAuthMode_DISABLED IAMAuthMode = "DISABLED"
	IAMAuthMode_REQUIRED IAMAuthMode = "REQUIRED"
)

type ReplicaMode string

const (
	ReplicaMode_open_read_only ReplicaMode = "open-read-only"
	ReplicaMode_mounted        ReplicaMode = "mounted"
)

type SourceType string

const (
	SourceType_db_instance           SourceType = "db-instance"
	SourceType_db_parameter_group    SourceType = "db-parameter-group"
	SourceType_db_security_group     SourceType = "db-security-group"
	SourceType_db_snapshot           SourceType = "db-snapshot"
	SourceType_db_cluster            SourceType = "db-cluster"
	SourceType_db_cluster_snapshot   SourceType = "db-cluster-snapshot"
	SourceType_custom_engine_version SourceType = "custom-engine-version"
	SourceType_db_proxy              SourceType = "db-proxy"
)

type TargetHealthReason string

const (
	TargetHealthReason_UNREACHABLE               TargetHealthReason = "UNREACHABLE"
	TargetHealthReason_CONNECTION_FAILED         TargetHealthReason = "CONNECTION_FAILED"
	TargetHealthReason_AUTH_FAILURE              TargetHealthReason = "AUTH_FAILURE"
	TargetHealthReason_PENDING_PROXY_CAPACITY    TargetHealthReason = "PENDING_PROXY_CAPACITY"
	TargetHealthReason_INVALID_REPLICATION_STATE TargetHealthReason = "INVALID_REPLICATION_STATE"
)

type TargetRole string

const (
	TargetRole_READ_WRITE TargetRole = "READ_WRITE"
	TargetRole_READ_ONLY  TargetRole = "READ_ONLY"
	TargetRole_UNKNOWN    TargetRole = "UNKNOWN"
)

type TargetState string

const (
	TargetState_REGISTERING TargetState = "REGISTERING"
	TargetState_AVAILABLE   TargetState = "AVAILABLE"
	TargetState_UNAVAILABLE TargetState = "UNAVAILABLE"
)

type TargetType string

const (
	TargetType_RDS_INSTANCE            TargetType = "RDS_INSTANCE"
	TargetType_RDS_SERVERLESS_ENDPOINT TargetType = "RDS_SERVERLESS_ENDPOINT"
	TargetType_TRACKED_CLUSTER         TargetType = "TRACKED_CLUSTER"
)

type WriteForwardingStatus string

const (
	WriteForwardingStatus_enabled   WriteForwardingStatus = "enabled"
	WriteForwardingStatus_disabled  WriteForwardingStatus = "disabled"
	WriteForwardingStatus_enabling  WriteForwardingStatus = "enabling"
	WriteForwardingStatus_disabling WriteForwardingStatus = "disabling"
	WriteForwardingStatus_unknown   WriteForwardingStatus = "unknown"
)
