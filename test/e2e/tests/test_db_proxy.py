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

"""Integration tests for the RDS API DBProxy resource
"""

import time

import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import condition
from e2e import db_proxy
from e2e.fixtures import k8s_secret
from e2e import tag
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = 'dbproxies'

DELETE_WAIT_AFTER_SECONDS = 120

# Time we wait after resource becoming available in RDS and checking the CR's
# Status has been updated.
CHECK_STATUS_WAIT_SECONDS = 60*4

MODIFY_WAIT_AFTER_SECONDS = 20

@pytest.fixture
def postgres_proxy(sts_client):
    db_proxy_id = random_suffix_name("my-test-proxy", 20)
    db_proxy_engine_family = "POSTGRESQL"
    # The IAM role and secrect below has a complext dependency chain and we can hard code it for now
    # It needs create one rds instance -> create aws secret manager service's secret based on it -> create IAM role based on this secret
    # I don't have a better way to fit this dependency chain in current rds controller yet, hence hard code it for now
    account_id = sts_client.get_caller_identity().get("Account")
    secret_arn = f"arn:aws:secretsmanager:us-west-2:{account_id}:secret:prod/ack/persistent/secret-hGHdOK"
    description = "proxy created by ack"

    replacements = REPLACEMENT_VALUES.copy()
    replacements["DB_PROXY_NAME"] = db_proxy_id
    replacements["DB_PROXY_ENGINE_FAMILY"] = db_proxy_engine_family
    replacements["IAM_ROLE_ARN"] = get_bootstrap_resources().RDSProxyRole.arn
    replacements["SECRET_ARN"] = secret_arn
    replacements["DESCRIPTION"] = description

    resource_data = load_rds_resource(
        "db_proxy",
        additional_replacements=replacements,
    )

    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        db_proxy_id, namespace="default",
    )

    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield (ref, cr)

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        db_proxy.wait_until_deleted(db_proxy_id)
    except:
        pass

@service_marker
@pytest.mark.canary
class TestDBProxy:
    def test_crud_postgresql_proxy(
        self,
        postgres_proxy
    ):
        (ref, cr) = postgres_proxy
        db_proxy_id = cr["spec"]["name"]

        assert cr is not None
        assert k8s.get_resource_exists(ref)
        assert 'status' in cr
        assert 'status' in cr['status']
        assert cr['status']['status'] in ['creating', 'available']

        db_proxy.wait_until(
            db_proxy_id,
            db_proxy.status_matches('available'),
        )

        time.sleep(CHECK_STATUS_WAIT_SECONDS)

        assert k8s.wait_on_condition(
            ref,
            condition.CONDITION_TYPE_RESOURCE_SYNCED,
            "True",
            wait_periods=5,
            period_length=3
        ), "DB proxy not synced"

        # Start testing tag for proxy
        latest = db_proxy.get(db_proxy_id)
        arn = latest['DBProxyArn']

        # now start delete db proxy
        k8s.delete_custom_resource(ref)

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        db_proxy.wait_until_deleted(db_proxy_id)
