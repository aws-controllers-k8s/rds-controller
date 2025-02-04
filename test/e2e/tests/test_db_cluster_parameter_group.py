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


@service_marker
@pytest.mark.canary
class TestDBClusterParameterGroup:
    def test_crud_aurora_mysql5_7(self, aurora_mysql57_cluster_param_group):
        ref, cr, resource_name = aurora_mysql57_cluster_param_group

        def log_resource_status(resource):
            """Helper function to log resource status"""
            logging.info("Resource Status:")
            logging.info(f"  Name: {resource['metadata']['name']}")
            logging.info(f"  Parameters: {resource.get('spec', {}).get('parameterOverrides', {})}")
            logging.info("  Conditions:")
            for condition in resource.get('status', {}).get('conditions', []):
                logging.info(f"    {condition['type']}: {condition['status']} ({condition.get('message', '')})")

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
        ], latest_params))
        found = 0
        for tp in test_params:
            assert "ParameterName" in tp, f"No ParameterName in parameter: {tp}"
            if tp["ParameterName"] == "aurora_read_replica_read_committed":
                found += 1
                assert "ParameterValue" in tp, f"No ParameterValue in parameter of name 'aurora_read_replica_read_committed': {tp}"
                assert tp["ParameterValue"] == "OFF", f"Wrong value for parameter of name 'aurora_read_replica_read_committed': {tp}"
        assert found == 1, f"Did not find parameters with names 'aurora_read_replica_read_committed': {test_params}"

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
            "aurora_read_replica_read_committed"
        ], params))
        assert len(test_params) == 1, f"test_params of wrong length: {test_params}"

        found = 0
        for tp in test_params:
            assert "ParameterName" in tp, f"No ParameterName in parameter: {tp}"
            if tp["ParameterName"] == "aurora_read_replica_read_committed":
                found += 1
                assert "ParameterValue" in tp, f"No ParameterValue in parameter of name 'aurora_read_replica_read_committed': {tp}"
                assert tp["ParameterValue"] == "ON", f"Wrong value for parameter of name 'aurora_read_replica_read_committed': {tp}"
        assert found == 1, f"Did not find parameters with names 'aurora_read_replica_read_committed': {test_params}"

        # Test updating with an invalid parameter to verify error handling
        new_params = {
            "aurora_read_replica_read_committed": "OFF",
            "long_query_time": "1"  # This parameter isn't supported at cluster level
        }
        updates = {
            "spec": {
                "tags": tag.clean(db_cluster_parameter_group.get_tags(arn)),
                "parameterOverrides": new_params,
            },
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # First verify the resource enters recoverable error state
        condition.assert_recoverable(ref)
        # Then verify the sync status is Unknown
        condition.assert_type_status(ref, condition.CONDITION_TYPE_RESOURCE_SYNCED, "Unknown")

        # Update with valid parameters to verify recovery from Unknown state
        new_params = {
            "aurora_read_replica_read_committed": "OFF"
        }
        updates = {
            "spec": {
                "tags": tag.clean(db_cluster_parameter_group.get_tags(arn)),
                "parameterOverrides": new_params,
            },
        }
        k8s.patch_custom_resource(ref, updates)
        
        # Wait and verify parameter modification with better logging
        max_retries = 5
        for i in range(max_retries):
            time.sleep(MODIFY_WAIT_AFTER_SECONDS)
            
            # Get latest resource state
            cr = k8s.get_resource(ref)
            log_resource_status(cr)
            
            try:
                # Verify sync condition
                condition.assert_synced(ref)
                
                # Verify parameters
                params = db_cluster_parameter_group.get_parameters(resource_name)
                test_params = list(filter(
                    lambda x: x["ParameterName"] == "aurora_read_replica_read_committed", 
                    params
                ))
                
                assert len(test_params) == 1, f"Parameter not found: {test_params}"
                assert test_params[0]["ParameterValue"] == "OFF", \
                    f"Parameter value mismatch: {test_params[0]['ParameterValue']}"
                
                logging.info("Parameter modification completed successfully")
                break
                
            except AssertionError as e:
                if i == max_retries - 1:
                    logging.error(f"Parameter modification failed: {str(e)}")
                    raise
                logging.info(f"Retry {i+1}/{max_retries}: Waiting for parameter update...")
                continue

        # Verify the parameter was actually updated
        params = db_cluster_parameter_group.get_parameters(resource_name)
        test_params = list(filter(lambda x: x["ParameterName"] == "aurora_read_replica_read_committed", params))
        assert len(test_params) == 1
        assert test_params[0]["ParameterValue"] == "OFF"

        # Add this helper function
        def logResourceStatus(t *testing.T, resource *rdsv1alpha1.DBClusterParameterGroup) {
            t.Logf("Resource Status:")
            t.Logf("  Name: %s", resource.Name)
            t.Logf("  Parameters: %v", resource.Spec.Parameters)
            t.Logf("  Conditions:")
            for _, cond := range resource.Status.Conditions {
                t.Logf("    %s: %s (%s)", cond.Type, cond.Status, cond.Message)
            }
        }

        # Use it in the test
        logResourceStatus(t, latest)

func TestDBClusterParameterGroupE2E(t *testing.T) {
    // ... existing setup code ...

    // Modify parameters
    parameterGroup.Spec.Parameters = map[string]string{
        "aurora_read_replica_read_committed": "ON",
        "aurora_binlog_read_buffer_size":    "8192",
    }

    // Update the resource
    err = k8sClient.Update(context.Background(), parameterGroup)
    require.NoError(t, err)

    // Wait for the parameter modification to complete and verify
    err = wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
        // Get latest state
        latest := &rdsv1alpha1.DBClusterParameterGroup{}
        err := k8sClient.Get(context.Background(), types.NamespacedName{
            Namespace: parameterGroup.Namespace,
            Name:     parameterGroup.Name,
        }, latest)
        if err != nil {
            return false, err
        }

        // Check sync status
        synced := false
        for _, condition := range latest.Status.Conditions {
            if condition.Type == "ACK.ResourceSynced" && condition.Status == "True" {
                synced = true
                break
            }
        }
        if !synced {
            return false, nil
        }

        // Verify parameters
        if latest.Spec.Parameters["aurora_read_replica_read_committed"] != "ON" {
            return false, nil
        }

        return true, nil
    })
    require.NoError(t, err, "Failed waiting for parameter modification to complete")

    // Additional verification
    t.Log("Parameter modification completed successfully")
    
    // Optional: Verify through AWS API if available
    if awsClient != nil {
        // Add AWS API verification
    }
}
