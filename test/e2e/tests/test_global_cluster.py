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
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import condition
from e2e import global_cluster
from e2e import tag

RESOURCE_PLURAL = 'globalclusters'

DELETE_WAIT_AFTER_SECONDS = 120

MODIFY_WAIT_AFTER_SECONDS = 20


@pytest.fixture
def postgresql_global_cluster():
    global_cluster_id = random_suffix_name("my-gc", 20)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["GLOBAL_CLUSTER_NAME"] = global_cluster_id
    replacements["GLOBAL_CLUSTER_ENGINE"] = "aurora-postgresql"
    replacements["GLOBAL_CLUSTER_DB_NAME"] = "testdb"

    resource_data = load_rds_resource(
        "global_cluster",
        additional_replacements=replacements,
    )

    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        global_cluster_id, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert 'status' in cr
    assert 'status' in cr['status']
    assert cr['status']['status'] == 'available'

    yield (ref, cr, global_cluster_id)

    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    global_cluster.wait_until_deleted(global_cluster_id)


@pytest.fixture
def tagged_global_cluster():
    global_cluster_id = random_suffix_name("my-gc-tags", 20)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["GLOBAL_CLUSTER_NAME"] = global_cluster_id
    replacements["GLOBAL_CLUSTER_ENGINE"] = "aurora-postgresql"
    replacements["GLOBAL_CLUSTER_DB_NAME"] = "testdb"

    resource_data = load_rds_resource(
        "global_cluster_tags",
        additional_replacements=replacements,
    )

    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        global_cluster_id, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert 'status' in cr
    assert 'status' in cr['status']
    assert cr['status']['status'] == 'available'

    yield (ref, cr, global_cluster_id)

    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    global_cluster.wait_until_deleted(global_cluster_id)


@service_marker
@pytest.mark.canary
class TestGlobalCluster:

    def test_crud_postgresql_globalcluster(
            self, postgresql_global_cluster,
    ):
        ref, cr, global_cluster_id = postgresql_global_cluster

        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'status' in cr
        assert 'status' in cr['status']
        condition.assert_synced(ref)

        latest = global_cluster.get(global_cluster_id)
        assert latest is not None
        assert 'GlobalClusterArn' in latest

    def test_crud_globalcluster_tags(
            self, tagged_global_cluster,
    ):
        ref, _, global_cluster_id = tagged_global_cluster

        condition.assert_synced(ref)

        latest = global_cluster.get(global_cluster_id)
        arn = latest['GlobalClusterArn']

        # Verify tags were applied on creation
        expect_tags = [
            {"Key": "environment", "Value": "dev"}
        ]
        latest_tags = tag.clean(global_cluster.get_tags(arn))
        assert expect_tags == latest_tags

        # Update tags
        new_tags = [
            {
                "key": "environment",
                "value": "prod",
            },
            {
                "key": "team",
                "value": "platform",
            },
        ]
        updates = {
            "spec": {"tags": new_tags},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        after_update_expected_tags = [
            {"Key": "environment", "Value": "prod"},
            {"Key": "team", "Value": "platform"},
        ]
        latest_tags = tag.clean(global_cluster.get_tags(arn))
        assert latest_tags == after_update_expected_tags
