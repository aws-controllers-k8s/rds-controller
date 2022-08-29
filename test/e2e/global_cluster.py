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

"""Utilities for working with Global Cluster resources"""

import datetime
import time
import typing

import boto3
import pytest

DEFAULT_WAIT_UNTIL_TIMEOUT_SECONDS = 60*10
DEFAULT_WAIT_UNTIL_INTERVAL_SECONDS = 15
DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 60*10
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15

GlobalClusterMatchFunc = typing.NewType(
    'GlobalClusterMatchFunc',
    typing.Callable[[dict], bool],
)

class StatusMatcher:
    def __init__(self, status):
        self.match_on = status

    def __call__(self, record: dict) -> bool:
        return 'Status' in record and record['Status'] == self.match_on


def status_matches(status: str) -> GlobalClusterMatchFunc:
    return StatusMatcher(status)


def wait_until(
        global_cluster_id: str,
        match_fn: GlobalClusterMatchFunc,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a DB global cluster with a supplied ID is returned from the RDS API
    and the matching functor returns True.

    Usage:
        from e2e.db_global_cluster import wait_until, status_matches

        wait_until(
            global_cluster_id,
            status_matches("available"),
        )

    Raises:
        pytest.fail upon timeout
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while not match_fn(get(global_cluster_id)):
        if datetime.datetime.now() >= timeout:
            pytest.fail("failed to match Global Cluster before timeout")
        time.sleep(interval_seconds)


def wait_until_deleted(
        global_cluster_id: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a Global Cluster with a supplied ID is no longer returned from
    the RDS API.

    Usage:
        from e2e.global_cluster import wait_until_deleted

        wait_until_deleted(global_cluster_id)

    Raises:
        pytest.fail upon timeout or if the Global Cluster goes to any other status
        other than 'deleting'
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for Global Cluster to be "
                "deleted in RDS API"
            )
        time.sleep(interval_seconds)

        latest = get(global_cluster_id)
        if latest is None:
            break

        if latest['Status'] != "deleting":
            pytest.fail(
                "Status is not 'deleting' for global cluster that was "
                "deleted. Status is " + latest['Status']
            )


def get(global_cluster_id):
    """Returns a dict containing the Global Cluster record from the RDS API.

    If no such Global Cluster exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.describe_global_clusters(GlobalClusterIdentifier=global_cluster_id)
        assert len(resp['GlobalClusters']) == 1
        return resp['GlobalClusters'][0]
    except c.exceptions.GlobalClusterNotFoundFault:
        return None


    
def get_tags(global_cluster_arn):
    """Returns a dict containing the Global Cluster's tag records from the RDS API.
    Currently RDS doesn't support add tags to global cluster, so this is a noop for now
    If no such global cluster exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.list_tags_for_resource(
            ResourceName=global_cluster_arn,
        )
        return resp['TagList']
    except c.exceptions.GlobalClusterNotFoundFault:
        return None
