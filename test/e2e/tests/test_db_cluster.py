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
from acktest.resources import random_suffix_name
from e2e import (CRD_GROUP, CRD_VERSION, condition, db_cluster,
                 load_rds_resource, service_marker, tag)
from e2e.fixtures import k8s_secret
from e2e.replacement_values import REPLACEMENT_VALUES

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

# MUP == Master user password...
MUP_NS = "default"
MUP_SEC_NAME = "dbclustersecrets"
MUP_SEC_KEY = "master_user_password"
MUP_SEC_VAL = "secretpass123456"
MUP_SEC_CLUSTER_NAME_PREFIX = "dbclustersecrets"


@pytest.fixture
def aurora_mysql_cluster(k8s_secret):
    db_cluster_id = random_suffix_name("my-aurora-mysql", 20)
    db_name = "mydb"
    secret = k8s_secret(
        MUP_NS,
        random_suffix_name(MUP_SEC_CLUSTER_NAME_PREFIX, 24),
        MUP_SEC_KEY,
        MUP_SEC_VAL,
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

    yield (ref, cr, db_cluster_id)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    db_cluster.wait_until_deleted(db_cluster_id)

@pytest.fixture(scope="module")
def aurora_postgres_cluster(k8s_secret):
    db_cluster_id = random_suffix_name("my-aurora-postgres", 32)
    secret = k8s_secret(
        MUP_NS,
        random_suffix_name("clustersecret", 32),
        MUP_SEC_KEY,
        MUP_SEC_VAL,
    )

    resource_data = load_rds_resource(
        "db_cluster_aurora_postgres",
        additional_replacements={
            "DB_CLUSTER_ID": db_cluster_id,
            "MASTER_USER_PASS_SECRET_NAMESPACE": secret.ns,
            "MASTER_USER_PASS_SECRET_NAME": secret.name,
            "MASTER_USER_PASS_SECRET_KEY": secret.key,
        },
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

    yield (ref, cr, db_cluster_id, secret.name)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    db_cluster.wait_until_deleted(db_cluster_id)

@pytest.fixture(scope="module")
def aurora_postgres_cluster_log_exports(k8s_secret):
    db_cluster_id = random_suffix_name("my-aurora-postgres-log-exports", 35)
    secret = k8s_secret(
        MUP_NS,
        f"{MUP_SEC_NAME}-postgres-{db_cluster_id}",
        MUP_SEC_KEY,
        MUP_SEC_VAL,
    )

    resource_data = load_rds_resource(
        "db_cluster_aurora_postgres_log_exports",
        additional_replacements={
            "DB_CLUSTER_ID": db_cluster_id,
            "MASTER_USER_PASS_SECRET_NAMESPACE": secret.ns,
            "MASTER_USER_PASS_SECRET_NAME": secret.name,
            "MASTER_USER_PASS_SECRET_KEY": secret.key,
        },
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

    yield (ref, cr, db_cluster_id)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    db_cluster.wait_until_deleted(db_cluster_id)

@service_marker
@pytest.mark.canary
class TestDBCluster:
    def test_crud_mysql_serverless(
            self, aurora_mysql_cluster,
    ):
        ref, cr, db_cluster_id = aurora_mysql_cluster

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
        latest_tags = tag.clean(db_cluster.get_tags(arn))
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
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        latest_tags = tag.clean(db_cluster.get_tags(arn))
        after_update_expected_tags = [
            {
                "Key": "environment",
                "Value": "prod",
            }
        ]
        assert latest_tags == after_update_expected_tags

    def test_flip_enable_iam_db_authn(
            self, aurora_postgres_cluster,
    ):
        ref, _, db_cluster_id, _ = aurora_postgres_cluster
        db_cluster.wait_until(
            db_cluster_id,
            db_cluster.status_matches('available'),
        )

        current = db_cluster.get(db_cluster_id)
        assert current is not None
        assert current["IAMDatabaseAuthenticationEnabled"] == False
        k8s.patch_custom_resource(
            ref,
            {"spec": {"enableIAMDatabaseAuthentication": True}},
        )

        db_cluster.wait_until(
            db_cluster_id,
            db_cluster.AttributeMatcher("IAMDatabaseAuthenticationEnabled", True),
        )

        latest = db_cluster.get(db_cluster_id)
        assert latest is not None
        assert latest["IAMDatabaseAuthenticationEnabled"] == True

    def test_enable_cloudwatch_logs_exports(
        self, aurora_postgres_cluster,
    ):
        ref, _, db_cluster_id, _ = aurora_postgres_cluster
        db_cluster.wait_until(
            db_cluster_id,
            db_cluster.status_matches('available'),
        )

        current = db_cluster.get(db_cluster_id)
        assert current is not None
        enabledCloudwatchLogsExports = current.get("EnabledCloudwatchLogsExports",None)
        assert enabledCloudwatchLogsExports is None
        k8s.patch_custom_resource(
            ref,
            {"spec": {"enableCloudwatchLogsExports": ["postgresql"]}},
        )

        db_cluster.wait_until(
            db_cluster_id,
            db_cluster.AttributeMatcher("EnabledCloudwatchLogsExports", ["postgresql"]),
        )

        latest = db_cluster.get(db_cluster_id)
        assert latest is not None
        assert latest["EnabledCloudwatchLogsExports"] == ["postgresql"]

    def test_disable_cloudwatch_logs_exports(
        self, aurora_postgres_cluster_log_exports,
    ):
        ref, _, db_cluster_id = aurora_postgres_cluster_log_exports
        db_cluster.wait_until(
            db_cluster_id,
            db_cluster.status_matches('available'),
        )

        current = db_cluster.get(db_cluster_id)
        assert current is not None

        enabledCloudwatchLogsExports = current.get("EnabledCloudwatchLogsExports", None)
        assert enabledCloudwatchLogsExports is not None
        assert enabledCloudwatchLogsExports == ["postgresql"]
        
        k8s.patch_custom_resource(
            ref,
            {"spec": {"enableCloudwatchLogsExports": []}},
        )

        db_cluster.wait_until(
            db_cluster_id,
            db_cluster.status_matches("available"),
        )

        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        latest = db_cluster.get(db_cluster_id)
        assert latest is not None

        enabledCloudwatchLogsExportsLatest = latest.get("EnabledCloudwatchLogsExports", None)
        assert enabledCloudwatchLogsExportsLatest is None

    def test_update_dbcluster_password(
            self, aurora_postgres_cluster, k8s_secret,
    ):
        ref, _, db_cluster_id, secret_name = aurora_postgres_cluster
        db_cluster.wait_until(
            db_cluster_id,
            db_cluster.status_matches('available'),
        )
        cr = k8s.get_resource(ref)
        assert cr is not None
        lastAppliedSecretRef = cr['metadata']['annotations']['rds.services.k8s.aws/last-applied-secret-reference']
        assert lastAppliedSecretRef == f"{MUP_NS}/{secret_name}.{MUP_SEC_KEY}"
        
        new_secret = k8s_secret(
            MUP_NS,
            f"{MUP_SEC_NAME}-postgres-{db_cluster_id}-2",
            MUP_SEC_KEY,
            MUP_SEC_VAL,
        )
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=20)

        updates = {
            "spec": {
                "masterUserPassword": {
                    "name": new_secret.name,
                    "namespace": new_secret.ns,
                    "key": new_secret.key,
                },
            },
        }
        
        k8s.patch_custom_resource(ref, updates)
        time.sleep(35)

        condition.assert_synced(ref)
        cr = k8s.get_resource(ref)
        
        assert cr is not None
        assert 'status' in cr
        assert 'status' in cr['status']
        assert cr['status']['status'] == 'available'

        lastAppliedSecretRef = cr['metadata']['annotations']['rds.services.k8s.aws/last-applied-secret-reference']
        assert lastAppliedSecretRef == f"{new_secret.ns}/{new_secret.name}.{new_secret.key}"

    def test_restore_cluster_to_latest_point_in_time(
            self, k8s_secret
    ):
        source_db_cluster_id = random_suffix_name("my-aurora-postgres-clone", 32)
        db_cluster_id = random_suffix_name("my-aurora-postgres-clone", 32)
        secret = k8s_secret(
            MUP_NS,
            random_suffix_name("clustersecret", 32),
            MUP_SEC_KEY,
            MUP_SEC_VAL,
        )
        resource_data = load_rds_resource(
            "db_cluster_aurora_postgres_clone",
            additional_replacements={
                    "DB_CLUSTER_ID": db_cluster_id,
                    "MASTER_USER_PASS_SECRET_NAMESPACE": secret.ns,
                    "MASTER_USER_PASS_SECRET_NAME": secret.name,
                    "MASTER_USER_PASS_SECRET_KEY": secret.key,
                    "SOURCE_DB_CLUSTER_ID": source_db_cluster_id,
                    "RESTORE_TYPE": "copy-on-write",
                    "USE_LATEST_RESTORABLE_TIME": "true",
            },
        )

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            db_cluster_id, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert 'status' in cr
        assert 'conditions' in cr['status']
        assert 'DBClusterNotFoundFault' in cr['status']['conditions'][0]['message']

        try:
            _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            assert deleted
            time.sleep(DELETE_WAIT_AFTER_SECONDS)
        except:
            pass

        db_cluster.wait_until_deleted(db_cluster_id)
