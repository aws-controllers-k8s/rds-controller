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

"""Integration tests for the RDS API DBClusterEndpoint resource
"""

import logging
import time

import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import db_cluster_endpoint
from e2e import tag
from e2e.fixtures import k8s_secret
from .test_db_cluster import aurora_mysql_cluster

RESOURCE_PLURAL = 'dbclusterendpoints'

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 120
DBINSTANCE_MAX_WAIT_FOR_SYNCED_SECONDS = 20

ENDPOINT_TYPE = "ANY"
UPDATED_ENDPOINT_TYPE = "READER"

@pytest.fixture
def db_cluster_endpoint_resource(aurora_mysql_cluster):
    cluster_ref, _, cluster_id = aurora_mysql_cluster

    assert k8s.wait_on_condition(cluster_ref, "ACK.ResourceSynced", "True", wait_periods=DBINSTANCE_MAX_WAIT_FOR_SYNCED_SECONDS)
    
    resource_name = random_suffix_name("custom-endpoint", 24)
    
    replacements = REPLACEMENT_VALUES.copy()
    replacements["DB_CLUSTER_ENDPOINT_IDENTIFIER"] = resource_name
    replacements["DB_CLUSTER_IDENTIFIER"] = cluster_id
    replacements["ENDPOINT_TYPE"] = ENDPOINT_TYPE
    
    resource_data = load_rds_resource(
        "db_cluster_endpoint",
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
    
    yield ref, cr, resource_name, cluster_id
    
    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass
    
    db_cluster_endpoint.wait_until_deleted(resource_name)


@service_marker
@pytest.mark.canary
class TestDBClusterEndpoint:
    def test_crud(self, db_cluster_endpoint_resource):
        ref, cr, resource_name, cluster_name = db_cluster_endpoint_resource
        
        # Let's check that the DB cluster endpoint appears in RDS
        latest = db_cluster_endpoint.get(resource_name)
        assert latest is not None
        assert latest['CustomEndpointType'] == ENDPOINT_TYPE
        assert latest['DBClusterIdentifier'] == cluster_name
        
        # Test initial tags are created
        initial_tags = [
            {"Key": "environment", "Value": "dev"}
        ]

        endpoint_arn = latest['DBClusterEndpointArn']
        latest_tags = tag.clean(db_cluster_endpoint.get_tags(endpoint_arn))
        assert initial_tags == latest_tags
        
        # Update the tags
        new_tags = [
            {"key": "environment", "value": "prod"},
            {"key": "application", "value": "testing"}
        ]
        updates = {
            "spec": {
                "endpointType": UPDATED_ENDPOINT_TYPE,
                "tags": new_tags
            }
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Verify tags were updated
        expected_tags = [
            {"Key": "environment", "Value": "prod"},
            {"Key": "application", "Value": "testing"}
        ]
        latest_tags = tag.clean(db_cluster_endpoint.get_tags(endpoint_arn))
        assert expected_tags == latest_tags

        latest = db_cluster_endpoint.get(resource_name)
        assert latest is not None
        assert latest['CustomEndpointType'] == UPDATED_ENDPOINT_TYPE
