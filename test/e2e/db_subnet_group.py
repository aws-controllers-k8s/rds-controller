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

"""Utilities for working with DB subnet resources"""

import datetime
import time

import boto3
import pytest

DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 60*10
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15


def wait_until_deleted(
        db_subnet_group_name: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a DB subnet_group with a supplied ID is no longer returned from
    the RDS API.

    Usage:
        from e2e.db_subnet_group import wait_until_deleted

        wait_until_deleted(subnet_group_id)

    Raises:
        pytest.fail upon timeout or if the DB subnet_group goes to any other status
        other than 'deleting'
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for DB subnet_group to be "
                "deleted in RDS API"
            )
        time.sleep(interval_seconds)

        latest = get(db_subnet_group_name)
        if latest is None:
            break

        if latest['DBSubnetGroupStatus'] != "deleting":
            pytest.fail(
                "Status is not 'deleting' for DB subnet_group that was "
                "deleted. Status is " + latest['DBSubnetGroupStatus']
            )


def get(db_subnet_group_name):
    """Returns a dict containing the DB subnet_group record from the RDS API.

    If no such DB subnet_group exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.describe_db_subnet_groups(
            DBSubnetGroupName=db_subnet_group_name,
        )
        assert len(resp['DBSubnetGroups']) == 1
        return resp['DBSubnetGroups'][0]
    except c.exceptions.DBSubnetGroupNotFoundFault:
        return None


def get_tags(db_subnet_group_arn):
    """Returns a dict containing the DB subnet group's tag records from the RDS
    API.

    If no such DB subnet group exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.list_tags_for_resource(
            ResourceName=db_subnet_group_arn,
        )
        return resp['TagList']
    except c.exceptions.DBSubnetGroupNotFoundFault:
        return None
