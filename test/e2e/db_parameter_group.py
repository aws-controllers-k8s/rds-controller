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

"""Utilities for working with DB parameter group resources"""

import datetime
import time

import boto3
import pytest

DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 60*10
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15


def wait_until_deleted(
        pg_name: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a DB param group with a supplied ID is no longer returned
    from the RDS API.

    Usage:
        from e2e.db_parameter_group import wait_until_deleted

        wait_until_deleted(pg_name)

    Raises:
        pytest.fail upon timeout or if the DB param group goes to any other
        status other than 'deleting'
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for DB param group to be "
                "deleted in RDS API"
            )
        time.sleep(interval_seconds)

        latest = get(pg_name)
        if latest is None:
            break


def get(db_parameter_group_name):
    """Returns a dict containing the DB parameter group record from the RDS
    API.

    If no such DB parameter group exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.describe_db_parameter_groups(
            DBParameterGroupName=db_parameter_group_name,
        )
        assert len(resp['DBParameterGroups']) == 1
        return resp['DBParameterGroups'][0]
    except c.exceptions.DBParameterGroupNotFoundFault:
        return None


def get_parameters(db_parameter_group_name):
    """Returns a dict containing the paramters of a given parameter group

    If no such DB parameter group exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.describe_db_parameters(
            DBParameterGroupName=db_parameter_group_name,
        )
        return resp['Parameters']
    except c.exceptions.DBParameterGroupNotFoundFault:
        return None


def get_engine_default_parameters(db_parameter_group_family):
    """Returns a dict containing the engine default parameters for a given parameter group family
    
    This function calls DescribeEngineDefaultParameters to get the default parameter metadata
    that's used as fallback validation in cluster parameter groups.
    """
    c = boto3.client('rds')
    try:
        all_parameters = []
        marker = None
        
        while True:
            if marker:
                resp = c.describe_engine_default_parameters(
                    DBParameterGroupFamily=db_parameter_group_family,
                    Marker=marker
                )
            else:
                resp = c.describe_engine_default_parameters(
                    DBParameterGroupFamily=db_parameter_group_family,
                )
            
            parameters = resp['EngineDefaults']['Parameters']
            all_parameters.extend(parameters)
            
            # Check if there are more results
            if 'Marker' in resp['EngineDefaults']:
                marker = resp['EngineDefaults']['Marker']
            else:
                break
                
        return all_parameters
    except Exception as e:
        return None


def get_tags(db_parameter_group_arn):
    """Returns a dict containing the DB parameter group's tag records from the
    RDS API.

    If no such DB parameter group exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.list_tags_for_resource(
            ResourceName=db_parameter_group_arn,
        )
        return resp['TagList']
    except c.exceptions.DBParameterGroupNotFoundFault:
        return None
