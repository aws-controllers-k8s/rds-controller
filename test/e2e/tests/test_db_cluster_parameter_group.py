# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for the RDS API DBClusterParameterGroup resource
"""

import logging
import time

import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import db_cluster_parameter_group
from e2e import db_parameter_group
from e2e import tag
from e2e import condition

RESOURCE_PLURAL = 'dbclusterparametergroups'

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
# NOTE(jaypipes): According to the RDS API documentation, updating tags can
# take several minutes before the new tag values are available due to caching.
MODIFY_WAIT_AFTER_SECONDS = 180

RESOURCE_DESC_AURORA_MYSQL57 = "Parameters for Aurora MySQL 5.7-compatible"


@pytest.fixture
def aurora_mysql57_cluster_param_group():
    resource_name = random_suffix_name("aurora-mysql-5-7", 24)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["DB_CLUSTER_PARAMETER_GROUP_NAME"] = resource_name
    replacements["DB_CLUSTER_PARAMETER_GROUP_DESC"] = RESOURCE_DESC_AURORA_MYSQL57

    resource_data = load_rds_resource(
        "db_cluster_parameter_group_aurora_mysql5.7",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield ref, cr, resource_name

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    db_cluster_parameter_group.wait_until_deleted(resource_name)


@pytest.fixture
def aurora_mysql80_logging_cluster_param_group():
    resource_name = random_suffix_name("aurora-mysql8-logging", 32)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["DB_CLUSTER_PARAMETER_GROUP_NAME"] = resource_name
    replacements["DB_CLUSTER_PARAMETER_GROUP_DESC"] = "Test MySQL logging parameters for Aurora MySQL 8.0"
    replacements["DB_CLUSTER_PARAMETER_GROUP_FAMILY"] = "aurora-mysql8.0"
    replacements["PARAM_SLOW_QUERY_LOG_VALUE"] = "1"
    replacements["PARAM_LONG_QUERY_TIME_VALUE"] = "10"
    replacements["PARAM_LOG_QUERIES_NOT_USING_INDEXES_VALUE"] = "1"

    resource_data = load_rds_resource(
        "db_cluster_parameter_group_aurora_mysql8.0_logging",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield ref, cr, resource_name

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    db_cluster_parameter_group.wait_until_deleted(resource_name)


@pytest.fixture
def aurora_pg14_mixed_source_cluster_param_group():
    resource_name = random_suffix_name("aurora-pg14-mixed", 24)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["DB_CLUSTER_PARAMETER_GROUP_NAME"] = resource_name
    replacements["DB_CLUSTER_PARAMETER_GROUP_DESC"] = "Test mixed source parameters"

    resource_data = load_rds_resource(
        "db_cluster_parameter_group_aurora_postgresql14_mixed_source",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield ref, cr, resource_name

    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    db_cluster_parameter_group.wait_until_deleted(resource_name)


@service_marker
@pytest.mark.canary
class TestDBClusterParameterGroup:
    def test_crud_aurora_mysql5_7(self, aurora_mysql57_cluster_param_group):
        ref, cr, resource_name = aurora_mysql57_cluster_param_group

        # Let's check that the DB cluster parameter group appears in RDS
        latest = db_cluster_parameter_group.get(resource_name)
        assert latest is not None
        assert latest['Description'] == RESOURCE_DESC_AURORA_MYSQL57

        arn = latest['DBClusterParameterGroupArn']
        expect_tags = [
            {"Key": "environment", "Value": "dev"}
        ]
        latest_tags = tag.clean(db_cluster_parameter_group.get_tags(arn))
        assert expect_tags == latest_tags

        latest_params = db_cluster_parameter_group.get_parameters(resource_name)
        test_params = list(filter(lambda x: x["ParameterName"] in [
            "aurora_read_replica_read_committed",
            "aurora_binlog_read_buffer_size",
        ], latest_params))
        found = 0
        for tp in test_params:
            assert "ParameterName" in tp, f"No ParameterName in parameter: {tp}"
            if tp["ParameterName"] == "aurora_binlog_read_buffer_size":
                found += 1
                assert "ParameterValue" in tp, f"No ParameterValue in parameter of name 'aurora_binlog_read_buffer_size': {tp}"
                assert tp["ParameterValue"] == "8192", f"Wrong value for parameter of name 'aurora_binlog_read_buffer_size': {tp}"
            elif tp["ParameterName"] == "aurora_read_replica_read_committed":
                found += 1
                assert "ParameterValue" in tp, f"No ParameterValue in parameter of name 'aurora_read_replica_read_committed': {tp}"
                assert tp["ParameterValue"] == "OFF", f"Wrong value for parameter of name 'aurora_read_replica_read_committed': {tp}"
        assert found == 2, f"Did not find parameters with names 'aurora_binlog_read_buffer_size' and 'aurora_read_replica_read_committed': {test_params}"

        # OK, now let's update the tag set and check that the tags are
        # updated accordingly.
        new_tags = [
            {
                "key": "environment",
                "value": "prod",
            }
        ]
        new_params = {
            "aurora_read_replica_read_committed": "ON",
            "aurora_binlog_read_buffer_size": "5242880",
        }
        updates = {
            "spec": {
                "tags": new_tags,
                "parameterOverrides": new_params,
            },
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        latest_tags = tag.clean(db_cluster_parameter_group.get_tags(arn))
        after_update_expected_tags = [
            {
                "Key": "environment",
                "Value": "prod",
            }
        ]
        assert latest_tags == after_update_expected_tags
        params = db_cluster_parameter_group.get_parameters(resource_name)
        test_params = list(filter(lambda x: x["ParameterName"] in [
            "aurora_read_replica_read_committed",
            "aurora_binlog_read_buffer_size"
        ], params))
        assert len(test_params) == 2, f"test_params of wrong length: {test_params}"

        found = 0
        for tp in test_params:
            assert "ParameterName" in tp, f"No ParameterName in parameter: {tp}"
            if tp["ParameterName"] == "aurora_binlog_read_buffer_size":
                found += 1
                assert "ParameterValue" in tp, f"No ParameterValue in parameter of name 'aurora_binlog_read_buffer_size': {tp}"
                assert tp["ParameterValue"] == "5242880", f"Wrong value for parameter of name 'aurora_binlog_read_buffer_size': {tp}"
            elif tp["ParameterName"] == "aurora_read_replica_read_committed":
                found += 1
                assert "ParameterValue" in tp, f"No ParameterValue in parameter of name 'aurora_read_replica_read_committed': {tp}"
                assert tp["ParameterValue"] == "ON", f"Wrong value for parameter of name 'aurora_read_replica_read_committed': {tp}"
        assert found == 2, f"Did not find parameters with names 'aurora_binlog_read_buffer_size' and 'aurora_read_replica_read_committed': {test_params}"

    def test_mysql_logging_parameters(self, aurora_mysql80_logging_cluster_param_group):
        ref, cr, resource_name = aurora_mysql80_logging_cluster_param_group

        latest = db_cluster_parameter_group.get(resource_name)
        assert latest is not None

        instance_defaults = db_parameter_group.get_engine_default_parameters("mysql8.0")
        assert instance_defaults is not None, "Failed to get instance-level engine defaults"
        
        fallback_params = list(filter(lambda x: x["ParameterName"] in [
            "slow_query_log",
            "long_query_time", 
            "log_queries_not_using_indexes",
        ], instance_defaults))
        
        # Log debug info about found parameters
        found_param_names = [p['ParameterName'] for p in fallback_params]
        logging.debug(f"Found fallback parameters: {found_param_names}")
        
        assert len(fallback_params) == 3, f"Expected 3 MySQL logging parameters in instance defaults, found {len(fallback_params)}: {found_param_names}"
        
        for param in fallback_params:
            assert "ParameterName" in param, f"Missing ParameterName in fallback parameter: {param}"
            assert "IsModifiable" in param, f"Missing IsModifiable in fallback parameter: {param}"
            assert "ApplyType" in param, f"Missing ApplyType in fallback parameter: {param}"

        assert 'status' in cr
        assert 'parameterOverrideStatuses' in cr['status']
        
        # Verify the parameter statuses show our MySQL logging parameters
        status_params = cr['status']['parameterOverrideStatuses']
        param_names = [p['parameterName'] for p in status_params]
        
        assert "slow_query_log" in param_names, f"slow_query_log parameter missing from status: {param_names}"
        assert "long_query_time" in param_names, f"long_query_time parameter missing from status: {param_names}"
        assert "log_queries_not_using_indexes" in param_names, f"log_queries_not_using_indexes parameter missing from status: {param_names}"

        # Additional wait for AWS RDS parameter propagation
        # RDS parameter changes can take 5-10 minutes to be fully visible via API
        logging.debug("Waiting additional time for AWS RDS parameter propagation...")
        time.sleep(MODIFY_WAIT_AFTER_SECONDS) 
        
        latest_params = db_cluster_parameter_group.get_user_defined_parameters(resource_name)
        assert latest_params is not None, "Failed to get user-defined cluster parameters"
        
        test_params = list(filter(lambda x: x["ParameterName"] in [
            "slow_query_log",
            "long_query_time", 
            "log_queries_not_using_indexes",
        ], latest_params))
        
        # Check initial parameter values
        expected_initial_values = {
            "slow_query_log": "1",
            "long_query_time": "10",
            "log_queries_not_using_indexes": "1"
        }
        
        for tp in test_params:
            param_name = tp["ParameterName"]
            assert param_name in expected_initial_values, f"Unexpected parameter: {param_name}"
            assert tp["ParameterValue"] == expected_initial_values[param_name], \
                f"Wrong value for {param_name}: expected {expected_initial_values[param_name]}, got {tp['ParameterValue']}"
        
        assert len(test_params) == len(expected_initial_values), \
            f"Expected {len(expected_initial_values)} parameters, found {len(test_params)}: {test_params}"

    def test_mixed_source_parameters(self, aurora_pg14_mixed_source_cluster_param_group):
        """Tests that parameters with different RDS source classifications
        (user, engine-default, system) are all correctly tracked when the user
        explicitly sets them. Also tests that removing parameters from the
        desired spec triggers a reset.

        This covers the bug where Source="user" filtering caused parameters
        like shared_preload_libraries (source=engine-default) and ssl
        (source=system) to be invisible to the controller.
        """
        ref, cr, resource_name = aurora_pg14_mixed_source_cluster_param_group

        latest = db_cluster_parameter_group.get(resource_name)
        assert latest is not None

        # All 6 parameters should be set in AWS regardless of source classification
        all_params = db_cluster_parameter_group.get_parameters(resource_name)
        expected_params = {
            "pgaudit.log": "none",
            "rds.force_ssl": "1",
            "rds.log_retention_period": "10080",
            "shared_preload_libraries": "pg_stat_statements",
            "ssl": "1",
            "ssl_min_protocol_version": "TLSv1.2",
        }

        for param_name, expected_value in expected_params.items():
            matching = [p for p in all_params if p["ParameterName"] == param_name]
            assert len(matching) == 1, f"Parameter {param_name} not found in AWS"
            assert matching[0].get("ParameterValue") == expected_value, \
                f"Wrong value for {param_name}: expected {expected_value}, got {matching[0].get('ParameterValue')}"

        # Wait for controller to reconcile and reach synced state
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # All 6 params should appear in parameterOverrideStatuses
        cr = k8s.get_resource(ref)
        assert 'status' in cr
        assert 'parameterOverrideStatuses' in cr['status']
        status_params = cr['status']['parameterOverrideStatuses']
        status_param_names = [p['parameterName'] for p in status_params]
        for param_name in expected_params:
            assert param_name in status_param_names, \
                f"{param_name} missing from parameterOverrideStatuses: {status_param_names}"

        # Resource should be synced (no perpetual diff)
        condition.assert_synced(ref)

        # Now remove 3 parameters (including engine-default and system sourced ones)
        # and verify they get reset
        reduced_params = {
            "pgaudit.log": "none",
            "rds.force_ssl": "1",
            "rds.log_retention_period": "10080",
        }
        updates = {
            "spec": {
                "parameterOverrides": reduced_params,
            },
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # Verify only 3 params remain in status
        cr = k8s.get_resource(ref)
        status_params = cr['status']['parameterOverrideStatuses']
        status_param_names = [p['parameterName'] for p in status_params]
        for param_name in reduced_params:
            assert param_name in status_param_names, \
                f"{param_name} missing from parameterOverrideStatuses after update"
        for removed_name in ["shared_preload_libraries", "ssl", "ssl_min_protocol_version"]:
            assert removed_name not in status_param_names, \
                f"{removed_name} should have been removed from parameterOverrideStatuses"

        condition.assert_synced(ref)
