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

"""Integration tests for the RDS API DBCluster resource
"""

import time

import pytest

from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e import condition
from e2e import db_cluster
from e2e import tag
from e2e.fixtures import k8s_secret

RESOURCE_PLURAL = 'dbclusters'

DELETE_WAIT_AFTER_SECONDS = 120

# Time we wait after resource becoming available in RDS and checking the CR's
# Status has been updated.
#
# NOTE(jaypipes): I have witnessed DB clusters transition from creating ->
# available -> creating again :(
#
# This is rare, but seems to be when RDS schedules a snapshot shortly after a
# DB cluster is created. The cluster gets to 'available', then RDS pauses the
# DB cluster, does some sort of snapshotting operation, then unpauses the DB
# cluster. It seems that during this unpausing operation, the DB cluster's
# status gets set back to 'creating' again.
#
# Unfortunately, because Aurora serverless doesn't emit Events for creation and
# deletion like it does for DB instances, this is difficult, if not impossible,
# to gather data on. Setting the check wait seconds here to 2 minutes since I
# believe this whole "pause-then-snapshot-then-unpause" thing only takes about
# a minute until the DB cluster status settles into an 'available' state.
CHECK_STATUS_WAIT_SECONDS = 60*2

MODIFY_WAIT_AFTER_SECONDS = 20


@service_marker
@pytest.mark.canary
class TestDBCluster:

    # MUP == Master user password...
    MUP_NS = "default"
    MUP_SEC_NAME = "dbclustersecrets"
    MUP_SEC_KEY = "master_user_password"
    MUP_SEC_VAL = "secretpass123456"

    def test_crud_mysql_serverless(
            self,
            k8s_secret,
    ):
        db_cluster_id = "my-aurora-mysql"
        db_name = "mydb"
        secret = k8s_secret(
            self.MUP_NS,
            self.MUP_SEC_NAME,
            self.MUP_SEC_KEY,
            self.MUP_SEC_VAL,
        )

        replacements = REPLACEMENT_VALUES.copy()
        replacements['COPY_TAGS_TO_SNAPSHOT'] = "False"
        replacements["DB_CLUSTER_ID"] = db_cluster_id
        replacements["DB_NAME"] = db_name
        replacements["MASTER_USER_PASS_SECRET_NAMESPACE"] = secret.ns
        replacements["MASTER_USER_PASS_SECRET_NAME"] = secret.name
        replacements["MASTER_USER_PASS_SECRET_KEY"] = secret.key

        resource_data = load_rds_resource(
            "db_cluster_mysql_serverless",
            additional_replacements=replacements,
        )

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            db_cluster_id, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert 'status' in cr
        assert 'status' in cr['status']
        assert cr['status']['status'] == 'creating'
        condition.assert_not_synced(ref)

        db_cluster.wait_until(
            db_cluster_id,
            db_cluster.status_matches('available'),
        )

        time.sleep(CHECK_STATUS_WAIT_SECONDS)

        # Before we update the DBCluster CR below, let's check to see that the
        # Status field in the CR has been updated to something other than
        # 'creating', which is what is set after the initial creation.  The
        # CR's `Status.Status` should be updated because the CR is requeued on
        # successful reconciliation loops and subsequent reconciliation loops
        # call ReadOne and should update the CR's Status with the latest
        # observed information.
        # https://github.com/aws-controllers-k8s/community/issues/923
        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'status' in cr
        assert 'status' in cr['status']
        assert cr['status']['status'] != 'creating'
        condition.assert_synced(ref)

        # We're now going to modify the CopyTagsToSnapshot field of the DB
        # instance, wait some time and verify that the RDS server-side resource
        # shows the new value of the field.
        latest = db_cluster.get(db_cluster_id)
        assert latest is not None
        assert latest['CopyTagsToSnapshot'] == False
        updates = {
            "spec": {"copyTagsToSnapshot": True},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        latest = db_cluster.get(db_cluster_id)
        assert latest is not None
        assert latest['CopyTagsToSnapshot'] == True

        arn = latest['DBClusterArn']
        expect_tags = [
            {"Key": "environment", "Value": "dev"}
        ]
        latest_tags = db_cluster.get_tags(arn)
        assert expect_tags == tag.cleaned(latest_tags)

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
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        latest_tags = db_cluster.get_tags(arn)
        after_update_expected_tags = [
            {
                "Key": "environment",
                "Value": "prod",
            }
        ]
        assert after_update_expected_tags == tag.cleaned(latest_tags)

        k8s.delete_custom_resource(ref)

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        db_cluster.wait_until_deleted(db_cluster_id)
