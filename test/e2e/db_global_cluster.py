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

"""Utilities for working with DB global cluster resources"""

import datetime
import time

import boto3
import pytest

DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 60*10
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15

def wait_until_deleted(
        global_cluster_id: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a DB global_cluster with a supplied ID is no longer returned from
    the RDS API.

    Usage:
        from e2e.db_global_cluster import wait_until_deleted

        wait_until_deleted(global_cluster_id)

    Raises:
        pytest.fail upon timeout or if the DB global_cluster goes to any other status
        other than 'deleting'
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for DB global_cluster to be "
                "deleted in RDS API"
            )
        time.sleep(interval_seconds)

        latest = get(global_cluster_id)
        if latest is None:
            break

        if latest['Status'] != "deleting":
            pytest.fail(
                "Status is not 'deleting' for DB global_cluster that was "
                "deleted. Status is " + latest['Status']
            )

def get(global_cluster_id):
    """Returns a dict containing the DB global cluster record from the RDS API.

    If no such DB global cluster exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.describe_global_clusters(GlobalClusterIdentifier=global_cluster_id)
        assert len(resp['GlobalClusters']) == 1
        return resp['GlobalClusters'][0]
    except c.exceptions.GlobalClusterNotFoundFault:
        return None