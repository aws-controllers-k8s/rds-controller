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

    # ... existing code ...

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
    
    # Wait and verify recovery from error state
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
            
            logging.info("Successfully recovered from error state and updated parameters")
            break
            
        except AssertionError as e:
            if i == max_retries - 1:
                logging.error(f"Failed to recover from error state: {str(e)}")
                raise
            logging.info(f"Retry {i+1}/{max_retries}: Waiting for recovery and parameter update...")
            continue
