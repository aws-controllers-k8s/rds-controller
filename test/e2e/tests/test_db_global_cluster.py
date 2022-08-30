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

"""Integration tests for the RDS API DBGlobalCluster resource
"""

import logging
import time

import pytest

from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e import condition
from e2e import db_global_cluster

RESOURCE_PLURAL = 'globalclusters'

DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 10

@service_marker
@pytest.mark.canary
class TestDBGlobalCluster:
    def test_crud(self):
        resource_name = "mygc-mysql"

        replacements = REPLACEMENT_VALUES.copy()
        replacements["DB_GLOBAL_CLUSTER_ID"] = resource_name

        resource_data = load_rds_resource(
            "db_global_cluster",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create the k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        time.sleep(90)
        
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)
        condition.assert_synced(ref)

        # Let's check that the DB global cluster appears in RDS
        latest = db_global_cluster.get(resource_name)
        assert latest is not None

        updates = {
            "spec": {"deletionProtection": False},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        latest = db_global_cluster.get(resource_name)
        assert latest["DeletionProtection"]

        # Delete the k8s resource on teardown of the module
        k8s.delete_custom_resource(ref)

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # DB global cluster should no longer appear in RDS
        latest = db_global_cluster.get(resource_name)
        assert latest is None
