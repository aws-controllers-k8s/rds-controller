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
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e import condition
from e2e import db_instance
from e2e.fixtures import k8s_secret

RESOURCE_PLURAL = 'dbinstances'

DELETE_WAIT_AFTER_SECONDS = 60*2

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
CHECK_STATUS_WAIT_SECONDS = 60*8

MODIFY_WAIT_AFTER_SECONDS = 20


@service_marker
@pytest.mark.canary
class TestDBInstance:

    # MUP == Master user password...
    MUP_NS = "default"
    MUP_SEC_NAME = "dbinstancesecrets"
    MUP_SEC_KEY = "master_user_password"
    MUP_SEC_VAL = "secretpass123456"

    def test_crud_postgres13_t3_micro(
            self,
            k8s_secret,
    ):
        db_instance_id = "pg13-t3-micro"
        secret = k8s_secret(
            self.MUP_NS,
            self.MUP_SEC_NAME,
            self.MUP_SEC_KEY,
            self.MUP_SEC_VAL,
        )

        replacements = REPLACEMENT_VALUES.copy()
        replacements['COPY_TAGS_TO_SNAPSHOT'] = "False"
        replacements["DB_INSTANCE_ID"] = db_instance_id
        replacements["MASTER_USER_PASS_SECRET_NAMESPACE"] = secret.ns
        replacements["MASTER_USER_PASS_SECRET_NAME"] = secret.name
        replacements["MASTER_USER_PASS_SECRET_KEY"] = secret.key

        resource_data = load_rds_resource(
            "db_instance_postgres13_t3_micro",
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
        assert 'status' in cr
        assert 'dbInstanceStatus' in cr['status']
        assert cr['status']['dbInstanceStatus'] == 'creating'
        condition.assert_not_synced(ref)

        db_instance.wait_until(
            db_instance_id,
            db_instance.status_matches('available'),
        )

        time.sleep(CHECK_STATUS_WAIT_SECONDS)

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
        assert latest['CopyTagsToSnapshot'] == False
        updates = {
            "spec": {"copyTagsToSnapshot": True},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        latest = db_instance.get(db_instance_id)
        assert latest is not None
        assert latest['CopyTagsToSnapshot'] == True

        k8s.delete_custom_resource(ref)

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        db_instance.wait_until_deleted(db_instance_id)
