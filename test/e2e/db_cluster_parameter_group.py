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

"""Utilities for working with DB cluster parameter group resources"""

import datetime
import time

import boto3
import pytest

DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 60*10
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15


def wait_until_deleted(
        cpg_name: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a DB cluster param group with a supplied ID is no longer
    returned from the RDS API.

    Usage:
        from e2e.db_cluster_parameter_group import wait_until_deleted

        wait_until_deleted(cpg_name)

    Raises:
        pytest.fail upon timeout
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for DB cluster param group to be "
                "deleted in RDS API"
            )
        time.sleep(interval_seconds)

        latest = get(cpg_name)
        if latest is None:
            break


def get(db_cluster_parameter_group_name):
    """Returns a dict containing the DB cluster parameter group record from the
    RDS API.

    If no such DB cluster parameter group exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.describe_db_cluster_parameter_groups(
            DBClusterParameterGroupName=db_cluster_parameter_group_name,
        )
        assert len(resp['DBClusterParameterGroups']) == 1
        return resp['DBClusterParameterGroups'][0]
    # NOTE(jaypipes): RDS DescribeDBClusterParameterGroups returns
    # DBParameterGroupNotFoundFault, *not* DBClusterParameterGroupNotFound.
    except c.exceptions.DBParameterGroupNotFoundFault:
        return None

def get_parameters(db_cluster_parameter_group_name):
    """Returns a dict containing the parameters of a given parameter group

    If no such DB cluster parameter group exists, returns empty list.
    """
    c = boto3.client('rds')
    try:
        all_parameters = []
        marker = None
        while True:
            params = {
                'DBClusterParameterGroupName': db_cluster_parameter_group_name,
            }
            if marker:
                params['Marker'] = marker
            resp = c.describe_db_cluster_parameters(**params)
            all_parameters.extend(resp['Parameters'])
            if 'Marker' in resp:
                marker = resp['Marker']
            else:
                break
        return all_parameters
    except c.exceptions.DBParameterGroupNotFoundFault:
        return []

def get_user_defined_parameters(db_cluster_parameter_group_name):
    """Returns a dict containing the user-defined parameters of a given cluster parameter group

    If no such DB cluster parameter group exists, returns empty list.
    Uses Source="user" to get only user-defined parameters (like the controller does).
    """
    c = boto3.client('rds')
    try:
        all_parameters = []
        marker = None
        
        while True:
            params = {
                'DBClusterParameterGroupName': db_cluster_parameter_group_name,
                'Source': 'user'
            }
            if marker:
                params['Marker'] = marker
                
            resp = c.describe_db_cluster_parameters(**params)
            all_parameters.extend(resp['Parameters'])
            
            # Check if there are more results
            if 'Marker' in resp:
                marker = resp['Marker']
            else:
                break
                
        return all_parameters
    except c.exceptions.DBParameterGroupNotFoundFault:
        return []

def get_tags(db_cluster_parameter_group_arn):
    """Returns a dict containing the DB cluster parameter group's tag records
    from the RDS API.

    If no such DB cluster parameter group exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.list_tags_for_resource(
            ResourceName=db_cluster_parameter_group_arn,
        )
        return resp['TagList']
    except c.exceptions.DBParameterGroupNotFoundFault:
        return None
