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

"""Integration tests for the RDS API OptionGroup resource
"""

import logging
import time

import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import option_group
from e2e import tag

RESOURCE_PLURAL = 'optiongroups'

CREATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10
MODIFY_WAIT_AFTER_SECONDS = 120

ENGINE_NAME = "mysql"
MAJOR_ENGINE_VERSION = "8.0"
OPTION_GROUP_DESCRIPTION = "Test option group for ACK integration tests"


@pytest.fixture
def option_group_resource():
    resource_name = random_suffix_name("ack-option-group", 24)

    replacements = REPLACEMENT_VALUES.copy()
    replacements["OPTION_GROUP_NAME"] = resource_name
    replacements["ENGINE_NAME"] = ENGINE_NAME
    replacements["MAJOR_ENGINE_VERSION"] = MAJOR_ENGINE_VERSION
    replacements["OPTION_GROUP_DESCRIPTION"] = OPTION_GROUP_DESCRIPTION

    resource_data = load_rds_resource(
        "option_group",
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
    time.sleep(CREATE_WAIT_AFTER_SECONDS)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield ref, cr, resource_name

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    option_group.wait_until_deleted(resource_name)


@service_marker
@pytest.mark.canary
class TestOptionGroup:
    def test_crud(self, option_group_resource):
        ref, cr, resource_name = option_group_resource

        # Let's check that the OptionGroup appears in RDS
        latest = option_group.get(resource_name)
        assert latest is not None
        assert latest['OptionGroupName'] == resource_name
        assert latest['EngineName'] == ENGINE_NAME
        assert latest['MajorEngineVersion'] == MAJOR_ENGINE_VERSION
        assert latest['OptionGroupDescription'] == OPTION_GROUP_DESCRIPTION

        # Test initial tags are created
        initial_tags = [
            {"Key": "environment", "Value": "dev"}
        ]

        option_group_arn = latest['OptionGroupArn']
        latest_tags = tag.clean(option_group.get_tags(option_group_arn))
        assert initial_tags == latest_tags

        # Update the tags
        new_tags = [
            {"key": "environment", "value": "prod"},
            {"key": "application", "value": "testing"}
        ]
        updates = {
            "spec": {
                "tags": new_tags
            }
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(MODIFY_WAIT_AFTER_SECONDS)
        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)

        # Verify tags were updated
        expected_tags = [
            {"Key": "environment", "Value": "prod"},
            {"Key": "application", "Value": "testing"}
        ]
        latest_tags = tag.clean(option_group.get_tags(option_group_arn))
        assert expected_tags == latest_tags

        # Verify the option group still exists and properties are unchanged
        latest = option_group.get(resource_name)
        assert latest is not None
        assert latest['OptionGroupName'] == resource_name
        assert latest['EngineName'] == ENGINE_NAME
        assert latest['MajorEngineVersion'] == MAJOR_ENGINE_VERSION
