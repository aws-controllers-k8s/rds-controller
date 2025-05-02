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

"""Integration tests for the RDS API DBInstance resource
"""

import time

import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import condition
from e2e import db_instance
from e2e.fixtures import k8s_secret
from e2e import tag

RESOURCE_PLURAL = 'dbinstances'

DELETE_WAIT_AFTER_SECONDS = 60*5

# Time we wait after resource becoming available in RDS and checking the CR's
# Status has been updated.
#
# NOTE(jaypipes): RDS does an automated backup as soon as a DB instance is
# created. This automated backup can take 2-3 minutes, during which time the DB
# instance's status will be 'backing-up'. In addition, I have noticed that
# sometimes RDS will reset master credentials after doing the initial snapshot
# backup, and this involves restarting the DB instance. This itself can take an
# additional 2-3 minutes.
#
# What this means is that the DB instance goes through the following status
# transitions:
#
# creating -> available -> backing-up -> available ->
# resetting-master-credentials -> restarting -> available
#
# This can take upwards of 7 minutes for the the DB instance to reach that
# "final" available state
MAX_WAIT_FOR_SYNCED_MINUTES = 20

MODIFY_WAIT_AFTER_SECONDS = 60

# MUP == Master user password...
MUP_NS = "default"
MUP_SEC_NAME_PREFIX = "dbinstancesecrets"
MUP_SEC_KEY = "master_user_password"
MUP_SEC_VAL = "secretpass123456"

@pytest.fixture
def postgres14_t3_micro_instance(k8s_secret):
    db_instance_id = random_suffix_name("pg14-t3-micro", 20)
    secret = k8s_secret(
        MUP_NS,
        random_suffix_name(MUP_SEC_NAME_PREFIX, 32),
        MUP_SEC_KEY,
        MUP_SEC_VAL,
    )

    replacements = REPLACEMENT_VALUES.copy()
    replacements['COPY_TAGS_TO_SNAPSHOT'] = "False"
    replacements["DB_INSTANCE_ID"] = db_instance_id
    replacements["MASTER_USER_PASS_SECRET_NAMESPACE"] = secret.ns
    replacements["MASTER_USER_PASS_SECRET_NAME"] = secret.name
    replacements["MASTER_USER_PASS_SECRET_KEY"] = secret.key

    resource_data = load_rds_resource(
        "db_instance_postgres14_t3_micro",
        additional_replacements=replacements,
    )

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        db_instance_id, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr, secret.name)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
    except:
        pass
    db_instance.wait_until_deleted(db_instance_id)

@service_marker
@pytest.mark.canary
class TestDBInstance:
    def test_crud_postgres14_t3_micro(
            self,
            postgres14_t3_micro_instance,
    ):
        (ref, cr, _) = postgres14_t3_micro_instance
        db_instance_id = cr["spec"]["dbInstanceIdentifier"]

        assert 'status' in cr
        assert 'dbInstanceStatus' in cr['status']
        assert cr['status']['dbInstanceStatus'] == 'creating'
        condition.assert_not_synced(ref)

        # Wait for the resource to get synced
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # After the resource is synced, assert that DBInstanceStatus is available
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert latest['DBInstanceStatus'] == 'available'
        assert latest['MultiAZ'] is False

        # Before we update the DBInstance CR below, let's check to see that the
        # DbInstanceStatus field in the CR has been updated to something other
        # than 'creating', which is what is set after the initial creation.
        # The CR's `Status.DBInstanceStatus` should be updated because the CR
        # is requeued on successful reconciliation loops and subsequent
        # reconciliation loops call ReadOne and should update the CR's Status
        # with the latest observed information.
        # https://github.com/aws-controllers-k8s/community/issues/923
        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'status' in cr
        assert 'dbInstanceStatus' in cr['status']
        assert cr['status']['dbInstanceStatus'] != 'creating'
        condition.assert_synced(ref)

        # We're now going to modify the CopyTagsToSnapshot field of the DB
        # instance, wait some time and verify that the RDS server-side resource
        # shows the new value of the field.
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert latest['CopyTagsToSnapshot'] is False
        updates = {
            "spec": {"copyTagsToSnapshot": True},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # wait for the resource to get synced after the patch
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # After resource is synced again, assert that patches are reflected in the AWS resource
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert latest['CopyTagsToSnapshot'] is True

        updates = {
            "spec": {"copyTagsToSnapshot": False},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # wait for the resource to get synced after the patch
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # After resource is synced again, assert that patches are reflected in the AWS resource
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert latest['CopyTagsToSnapshot'] is False

        arn = latest['DBInstanceArn']
        expect_tags = [
            {"Key": "environment", "Value": "dev"}
        ]
        latest_tags = tag.clean(db_instance.get_tags(arn))
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

        latest_tags = tag.clean(db_instance.get_tags(arn))
        after_update_expected_tags = [
            {
                "Key": "environment",
                "Value": "prod",
            }
        ]
        assert latest_tags == after_update_expected_tags

    def test_crud_postgres14_update_password(
            self,
            postgres14_t3_micro_instance,
            k8s_secret,
    ):
        (ref, cr, secret_name) = postgres14_t3_micro_instance
        db_instance_id = cr["spec"]["dbInstanceIdentifier"]

        assert 'status' in cr
        assert 'dbInstanceStatus' in cr['status']
        assert cr['status']['dbInstanceStatus'] == 'creating'
        condition.assert_not_synced(ref)

        # Assert that the last-applied-secret-reference annotation is set
        assert 'metadata' in cr
        assert 'annotations' in cr['metadata']
        assert 'rds.services.k8s.aws/last-applied-secret-reference' in cr['metadata']['annotations']
        lastAppliedSecretRef = cr['metadata']['annotations']['rds.services.k8s.aws/last-applied-secret-reference']
        assert lastAppliedSecretRef == f"{MUP_NS}/{secret_name}.{MUP_SEC_KEY}"

        # Wait for the resource to get synced
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # After the resource is synced, assert that DBInstanceStatus is available
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert latest['DBInstanceStatus'] == 'available'
        assert latest['MultiAZ'] is False

        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'status' in cr
        assert 'dbInstanceStatus' in cr['status']
        assert cr['status']['dbInstanceStatus'] != 'creating'
        condition.assert_synced(ref)

        # Let's now update the DBInstance secret and check that the CR's
        # `Status.DBInstanceStatus` is updated to 'resetting-master-credentials'
        new_secret = k8s_secret(
            MUP_NS,
            random_suffix_name(MUP_SEC_NAME_PREFIX, 32),
            MUP_SEC_KEY,
            MUP_SEC_VAL+"new",
        )
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
        condition.assert_not_synced(ref)
        cr = k8s.get_resource(ref)
        assert cr is not None
        assert 'status' in cr
        assert 'dbInstanceStatus' in cr['status']
        assert cr['status']['dbInstanceStatus'] == 'resetting-master-credentials'

        lastAppliedSecretRef = cr['metadata']['annotations']['rds.services.k8s.aws/last-applied-secret-reference']
        assert lastAppliedSecretRef == f"{new_secret.ns}/{new_secret.name}.{new_secret.key}"

    def test_enable_pi_postgres14_t3_micro(
            self,
            postgres14_t3_micro_instance,
    ):
        (ref, cr, _) = postgres14_t3_micro_instance
        db_instance_id = cr["spec"]["dbInstanceIdentifier"]

        assert 'status' in cr
        assert 'dbInstanceStatus' in cr['status']
        assert cr['status']['dbInstanceStatus'] == 'creating'
        condition.assert_not_synced(ref)

        # Wait for the resource to get synced
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # After the resource is synced, assert that DBInstanceStatus is available
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert latest['DBInstanceStatus'] == 'available'
        assert latest['MultiAZ'] is False

        # We're now going to enable the PerformanceInsights, which should force 
        # instance to change out of "available" status and temporarily make it
        # not synced
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert latest['PerformanceInsightsEnabled'] is False
        updates = {
            "spec": {"performanceInsightsEnabled": True},
        }
        k8s.patch_custom_resource(ref, updates)
        # Wait less time, so we can see it moving to ResourceSynced = False
        time.sleep(5)

        # Ensure the controller properly detects the status change
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "False", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # The resource should eventually come back into ResourceSynced = True
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # After resource is synced again, assert that patches are reflected in the AWS resource
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert latest['PerformanceInsightsEnabled'] is True

        # TODO: Ensure that the server side defaults
        # (PerformanceInsightsRetentionPeriod and PerformanceInsightsKMSKeyID)
        # are also persisted back into the spec. This currently does not work

    def test_state_field_flapping(
            self,
            postgres14_t3_micro_instance,
    ):
        """
        This test ensures that various state fields -- such as instanceClass --
        do not "flap" when updated between the original and new values.

        :see https://github.com/aws-controllers-k8s/community/issues/1773
        :see https://github.com/aws-controllers-k8s/community/issues/1650
        :see https://github.com/aws-controllers-k8s/community/issues/1376
        """
        (ref, cr, _) = postgres14_t3_micro_instance
        db_instance_id = cr["spec"]["dbInstanceIdentifier"]

        # latest CR should have desired state set properly...
        cr = k8s.get_resource(ref)
        assert cr["spec"]["dbInstanceClass"] == "db.t3.micro"
        assert not bool(cr["spec"]["multiAZ"])

        # Wait for the resource to get synced
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # We're now going to modify the instanceClass field of the DB instance,
        # wait some time and verify that the RDS server-side resource shows the
        # new value of the field.
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert 'DBInstanceClass' in latest
        assert latest['DBInstanceClass'] == "db.t3.micro"
        assert 'MultiAZ' in latest
        assert bool(latest['MultiAZ']) == False

        updates = {
            "spec": {"dbInstanceClass": "db.m5.large"},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # wait for the resource to get synced after the patch
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # latest CR should have desired state set properly...
        cr = k8s.get_resource(ref)
        assert cr["spec"]["dbInstanceClass"] == "db.m5.large"
        assert not bool(cr["spec"]["multiAZ"])

        # After resource is synced again, assert that patches are reflected in the AWS resource
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert 'DBInstanceClass' in latest
        if latest['DBInstanceClass'] != "db.m5.large":
            assert "PendingModifiedValues" in latest
            assert "DBInstanceClass" in latest["PendingModifiedValues"]
            assert latest["PendingModifiedValues"]["DBInstanceClass"] == "db.m5.large"
        assert 'MultiAZ' in latest
        assert bool(latest['MultiAZ']) == False

        # Now update the MultiAZ flag to True, wait and check that the MultiAZ
        # field value is set to True and that the DBInstanceClass hasn't
        # changed either.

        updates = {
            "spec": {"multiAZ": True},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        # wait for the resource to get synced after the patch
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES)

        # latest CR should have desired state set properly...
        cr = k8s.get_resource(ref)
        assert cr["spec"]["dbInstanceClass"] == "db.m5.large"
        assert bool(cr["spec"]["multiAZ"])

        # After resource is synced again, assert that patches are reflected in the AWS resource
        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert 'DBInstanceClass' in latest
        if latest['DBInstanceClass'] != "db.m5.large":
            assert "PendingModifiedValues" in latest
            assert "DBInstanceClass" in latest["PendingModifiedValues"]
            assert latest["PendingModifiedValues"]["DBInstanceClass"] == "db.m5.large"
        assert 'MultiAZ' in latest
        if not bool(latest['MultiAZ']):
            assert "PendingModifiedValues" in latest
            assert "MultiAZ" in latest["PendingModifiedValues"]
            assert bool(latest["PendingModifiedValues"]["MultiAZ"])
