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

"""Integration tests for the RDS API Global Cluster resource
"""

import time

import pytest

from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import condition
from e2e import global_cluster
from e2e.fixtures import k8s_secret
from e2e import tag
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = 'globalclusters'

DELETE_WAIT_AFTER_SECONDS = 120

# Time we wait after resource becoming available in RDS and checking the CR's
# Status has been updated.
CHECK_STATUS_WAIT_SECONDS = 60*4

MODIFY_WAIT_AFTER_SECONDS = 20


@service_marker
@pytest.mark.canary
class TestGlobalCluster:

    def test_crud_postgresql_globalcluster(
            self,
    ):
        global_cluster_id = "my-test-global-cluster"
        global_cluster_engine = "aurora-postgresql"
        global_cluster_db_name = 'testdb'

        replacements = REPLACEMENT_VALUES.copy()
        replacements["GLOBAL_CLUSTER_NAME"] = global_cluster_id
        replacements["GLOBAL_CLUSTER_ENGINE"] = global_cluster_engine
        replacements["GLOBAL_CLUSTER_DB_NAME"] = global_cluster_db_name

        resource_data = load_rds_resource(
            "global_cluster",
            additional_replacements=replacements,
        )

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            global_cluster_id, namespace="default",
        )
        # First try create global cluster 
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        # global cluster is available immediately upon created
        assert cr is not None
        assert 'status' in cr
        assert 'status' in cr['status']
        assert cr['status']['status'] == 'available'

        # assert global cluster is synced
        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'status' in cr
        assert 'status' in cr['status']
        condition.assert_ready(ref)

        latest = global_cluster.get(global_cluster_id)
        arn = latest['GlobalClusterArn']

        # now start delete global cluster
        k8s.delete_custom_resource(ref)

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        global_cluster.wait_until_deleted(global_cluster_id)
