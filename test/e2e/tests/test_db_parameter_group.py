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

"""Integration tests for the RDS API DBParameterGroup resource
"""

import logging
import time

import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import condition
from e2e import db_parameter_group
from e2e import tag

RESOURCE_PLURAL = 'dbparametergroups'

CREATE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@service_marker
@pytest.mark.canary
class TestDBParameterGroup:
    def test_create_delete_postgres13_standard(self):
        resource_name = random_suffix_name("pg13-standard", 32)
        resource_desc = "Parameters for PostgreSQL 13"

        replacements = REPLACEMENT_VALUES.copy()
        replacements["DB_PARAMETER_GROUP_NAME"] = resource_name
        replacements["DB_PARAMETER_GROUP_DESC"] = resource_desc

        resource_data = load_rds_resource(
            "db_parameter_group_postgres13_standard",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create the k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        time.sleep(CREATE_WAIT_AFTER_SECONDS)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        # Let's check that the DB parameter group appears in RDS
        latest = db_parameter_group.get(resource_name)
        assert latest is not None
        assert latest['Description'] == resource_desc

        params = db_parameter_group.get_parameters(resource_name)
        test_params = list(filter(lambda x: x["ParameterName"] in ["array_nulls", "authentication_timeout"], params))
        assert len(test_params) == 2
        assert test_params[0]["ParameterName"] == "array_nulls"
        assert test_params[0]["ParameterValue"] == "1"
        assert test_params[1]["ParameterName"] == "authentication_timeout"
        assert test_params[1]["ParameterValue"] == "50"

        arn = latest['DBParameterGroupArn']
        expect_tags = [
            {"Key": "environment", "Value": "dev"}
        ]
        latest_tags = tag.clean(db_parameter_group.get_tags(arn))
        assert expect_tags == latest_tags

        # OK, now let's update the tag set and check that the tags are
        # updated accordingly.
        new_tags = [
            {
                "key": "environment",
                "value": "prod",
            }
        ]
        new_params = {
            "array_nulls": "1",
            "authentication_timeout": "60",
        }
        
        updates = {
            "spec": {
                "tags": new_tags,
                "parameterOverrides": new_params,
            },
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)

        latest_tags = tag.clean(db_parameter_group.get_tags(arn))
        after_update_expected_tags = [
            {
                "Key": "environment",
                "Value": "prod",
            }
        ]
        assert latest_tags == after_update_expected_tags
        params = db_parameter_group.get_parameters(resource_name)
        test_params = list(filter(lambda x: x["ParameterName"] in ["array_nulls", "authentication_timeout"], params))
        assert len(test_params) == 2
        assert test_params[1]["ParameterName"] == "authentication_timeout"
        assert test_params[1]["ParameterValue"] == "60"

        # Delete the k8s resource on teardown of the module
        k8s.delete_custom_resource(ref)

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # DB parameter group should no longer appear in RDS
        latest = db_parameter_group.get(resource_name)
        assert latest is None
