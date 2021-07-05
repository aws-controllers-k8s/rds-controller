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

RESOURCE_PLURAL = 'dbclusterparametergroups'

DELETE_WAIT_AFTER_SECONDS = 10


@pytest.fixture(scope="module")
def rds_client():
    return boto3.client('rds')


@service_marker
@pytest.mark.canary
class TestDBClusterParameterGroup:
    def test_create_delete_aurora_mysql5_7(self, rds_client):
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
        aws_res = rds_client.describe_db_cluster_parameter_groups(
            DBClusterParameterGroupName=resource_name,
        )
        assert aws_res is not None
        assert len(aws_res['DBClusterParameterGroups']) == 1

        # Delete the k8s resource on teardown of the module
        k8s.delete_custom_resource(ref)

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # DB cluster parameter group should no longer appear in RDS
        try:
            aws_res = rds_client.describe_db_cluster_parameter_groups(
                DBClusterParameterGroupName=resource_name,
            )
            assert False
        # NOTE(jaypipes): RDS DescribeDBClusterParameterGroups returns
        # DBParameterGroupNotFoundFault, *not* DBClusterParameterGroupNotFound.
        except rds_client.exceptions.DBParameterGroupNotFoundFault:
            pass
