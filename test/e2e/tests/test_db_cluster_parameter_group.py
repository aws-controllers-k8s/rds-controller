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

"""Integration tests for the RDS API DBClusterParameterGroup resource
"""

import logging
import time

import pytest

from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import db_cluster_parameter_group
from e2e import tag
from e2e import condition

RESOURCE_PLURAL = 'dbclusterparametergroups'

DELETE_WAIT_AFTER_SECONDS = 10
# NOTE(jaypipes): According to the RDS API documentation, updating tags can
# take several minutes before the new tag values are available due to caching.
MODIFY_WAIT_AFTER_SECONDS = 180


@service_marker
@pytest.mark.canary
class TestDBClusterParameterGroup:
    def test_create_delete_aurora_mysql5_7(self):
        resource_name = "aurora-mysql-5-7"
        resource_desc = "Parameters for Aurora MySQL 5.7-compatible"

        replacements = REPLACEMENT_VALUES.copy()
        replacements["DB_CLUSTER_PARAMETER_GROUP_NAME"] = resource_name
        replacements["DB_CLUSTER_PARAMETER_GROUP_DESC"] = resource_desc

        resource_data = load_rds_resource(
            "db_cluster_parameter_group_aurora_mysql5.7",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create the k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        # Let's check that the DB cluster parameter group appears in RDS
        latest = db_cluster_parameter_group.get(resource_name)
        assert latest is not None
        assert latest['Description'] == resource_desc

        arn = latest['DBClusterParameterGroupArn']
        expect_tags = [
            {"Key": "environment", "Value": "dev"}
        ]
        latest_tags = tag.clean(db_cluster_parameter_group.get_tags(arn))
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

        latest_tags = tag.clean(db_cluster_parameter_group.get_tags(arn))
        after_update_expected_tags = [
            {
                "Key": "environment",
                "Value": "prod",
            }
        ]
        assert latest_tags == after_update_expected_tags

        # Delete the k8s resource on teardown of the module
        k8s.delete_custom_resource(ref)

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # DB cluster parameter group should no longer appear in RDS
        latest = db_cluster_parameter_group.get(resource_name)
        assert latest is None
