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

import boto3
import datetime
import logging
import time
from typing import Dict

import pytest

from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = 'dbinstances'

DELETE_WAIT_INTERVAL_SLEEP_SECONDS = 15
DELETE_WAIT_AFTER_SECONDS = 120
# NOTE(jaypipes): I've seen this take upwards of 5 minutes or more to fully see
# the DB instance not appear in the DescribeDBInstances call once
# DeleteDBInstance has been called (even with SkipFinalSnapshot=true)
DELETE_TIMEOUT_SECONDS = 600

CREATE_INTERVAL_SLEEP_SECONDS = 15
# Time to wait before we get to an expected `available` state.
CREATE_TIMEOUT_SECONDS = 600
# Time we wait after resource becoming available in RDS and checking the CR's
# Status has been updated
CHECK_STATUS_WAIT_SECONDS = 10

MODIFY_WAIT_AFTER_SECONDS = 20

@pytest.fixture(scope="module")
def rds_client():
    return boto3.client('rds')


@pytest.fixture(scope="module")
def master_user_pass_secret():
    ns = "default"
    name = "dbinstancesecrets"
    key = "master_user_password"
    secret_val = "secretpass123456"
    k8s.create_opaque_secret(ns, name, key, secret_val)
    yield ns, name, key
    k8s.delete_secret(ns, name)


@service_marker
@pytest.mark.canary
class TestDBInstance:
    def test_crud_postgres13_t3_micro(
            self,
            rds_client,
            master_user_pass_secret,
    ):
        db_id = "pg13-t3-micro"
        mup_sec_ns, mup_sec_name, mup_sec_key = master_user_pass_secret

        replacements = REPLACEMENT_VALUES.copy()
        replacements['COPY_TAGS_TO_SNAPSHOT'] = "False"
        replacements["DB_INSTANCE_ID"] = db_id
        replacements["MASTER_USER_PASS_SECRET_NAMESPACE"] = mup_sec_ns
        replacements["MASTER_USER_PASS_SECRET_NAME"] = mup_sec_name
        replacements["MASTER_USER_PASS_SECRET_KEY"] = mup_sec_key

        resource_data = load_rds_resource(
            "db_instance_postgres13_t3_micro",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create the k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            db_id, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert 'status' in cr
        assert 'dbInstanceStatus' in cr['status']
        assert cr['status']['dbInstanceStatus'] == 'creating'

        # Let's check that the DB instance appears in RDS
        aws_res = rds_client.describe_db_instances(DBInstanceIdentifier=db_id)
        assert aws_res is not None
        assert len(aws_res['DBInstances']) == 1
        dbi_rec = aws_res['DBInstances'][0]

        now = datetime.datetime.now()
        timeout = now + datetime.timedelta(seconds=CREATE_TIMEOUT_SECONDS)

        # TODO(jaypipes): Move this into generic AWS-side waiter
        while dbi_rec['DBInstanceStatus'] != "available":
            if datetime.datetime.now() >= timeout:
                pytest.fail("failed to find available DBInstance before timeout")
            time.sleep(CREATE_INTERVAL_SLEEP_SECONDS)
            aws_res = rds_client.describe_db_instances(DBInstanceIdentifier=db_id)
            assert len(aws_res['DBInstances']) == 1
            dbi_rec = aws_res['DBInstances'][0]

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

        # We're now going to modify the CopyTagsToSnapshot field of the DB
        # instance, wait some time and verify that the RDS server-side resource
        # shows the new value of the field.
        assert dbi_rec['CopyTagsToSnapshot'] == False
        updates = {
            "spec": {"copyTagsToSnapshot": True},
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        aws_res = rds_client.describe_db_instances(DBInstanceIdentifier=db_id)
        assert aws_res is not None
        assert len(aws_res['DBInstances']) == 1
        dbi_rec = aws_res['DBInstances'][0]
        assert dbi_rec['CopyTagsToSnapshot'] == True

        # Delete the k8s resource on teardown of the module
        k8s.delete_custom_resource(ref)

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        now = datetime.datetime.now()
        timeout = now + datetime.timedelta(seconds=DELETE_TIMEOUT_SECONDS)

        # DB instance should no longer appear in RDS
        while True:
            if datetime.datetime.now() >= timeout:
                pytest.fail("Timed out waiting for DB instance to being deleted in RDS API")
            time.sleep(DELETE_WAIT_INTERVAL_SLEEP_SECONDS)

            try:
                aws_res = rds_client.describe_db_instances(DBInstanceIdentifier=db_id)
                assert len(aws_res['DBInstances']) == 1
                dbi_rec = aws_res['DBInstances'][0]
                if dbi_rec['DBInstanceStatus'] != "deleting":
                    pytest.fail("DBInstanceStatus is not 'deleting' for DB instance that was deleted. DBInstanceStatus is "+dbi_rec['DBInstanceStatus'])
            except rds_client.exceptions.DBInstanceNotFoundFault:
                break
