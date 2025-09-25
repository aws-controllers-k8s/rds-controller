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

"""Integration tests for the RDS API DBClusterSnapshot resource
"""

import time

import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import condition
from e2e import db_cluster_snapshot
from e2e.fixtures import k8s_secret
from e2e import tag
from .test_db_cluster import aurora_mysql_cluster

RESOURCE_PLURAL = 'dbclustersnapshots'
MAX_WAIT_FOR_SYNCED_MINUTES = 20
DBINSTANCE_MAX_WAIT_FOR_SYNCED_MINUTES = 20

@pytest.fixture
def aurora_mysql_db_cluster_snapshot(aurora_mysql_cluster):
    (ref, cr, _) = aurora_mysql_cluster

    # Wait for the dbinstance to get synced
    assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=DBINSTANCE_MAX_WAIT_FOR_SYNCED_MINUTES)

    db_cluster_id = cr["spec"]["dbClusterIdentifier"]
    db_cluster_snapshot_id = random_suffix_name("cluster-snapshot", 20)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["DB_CLUSTER_ID"] = db_cluster_id
    replacements["DB_CLUSTER_SNAPSHOT_ID"] = db_cluster_snapshot_id

    resource_data = load_rds_resource(
        "db_cluster_snapshot",
        additional_replacements=replacements,
    )

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        db_cluster_snapshot_id, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
    except:
        pass
    db_cluster_snapshot.wait_until_deleted(db_cluster_snapshot_id)

@service_marker
@pytest.mark.canary
class TestDBClusterSnapshot:
    def test_cluster_snapshot_create_delete(
            self,
            aurora_mysql_db_cluster_snapshot,
    ):
        (ref, cr) = aurora_mysql_db_cluster_snapshot

        assert 'status' in cr
        assert 'status' in cr['status']
        assert cr['status']['status'] == 'creating'
        condition.assert_not_ready(ref)

        db_cluster_snapshot_id = cr["spec"]["dbClusterSnapshotIdentifier"]

        # Wait for the resource to get synced
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # After the resource is synced, assert that DBSnapshotStatus is available
        latest = db_cluster_snapshot.get(db_cluster_snapshot_id)
        assert latest is not None
        assert latest['Status'] == 'available'

        # wait for the resource to get synced after the patch
        assert k8s.wait_on_condition(ref, "Ready", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        arn = latest['DBClusterSnapshotArn']
        expect_tags = [
            {"Key": "environment", "Value": "dev"}
        ]
        latest_tags = tag.clean(db_cluster_snapshot.get_tags(arn))
        assert expect_tags == latest_tags

        # OK, now let's update the tag set and check that the tags are
        # updated accordingly.
        new_tags = [
            {
                "key": "environment",
                "value": "prod",
            }
        ]
        updates = {
            "spec": {"tags": new_tags},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(15)

        latest_tags = tag.clean(db_cluster_snapshot.get_tags(arn))
        after_update_expected_tags = [
            {
                "Key": "environment",
                "Value": "prod",
            }
        ]
        assert latest_tags == after_update_expected_tags
