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

"""Utilities for working with DB cluster resources"""

import datetime
import time
import typing
import logging
import botocore.exceptions

import boto3
import pytest
from e2e.retry_util import retry_on_api_error

DEFAULT_WAIT_UNTIL_TIMEOUT_SECONDS = 60*20  # Increased from 60*10
DEFAULT_WAIT_UNTIL_INTERVAL_SECONDS = 15
DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 60*20  # Increased from 60*10
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15

ClusterMatchFunc = typing.NewType(
    'ClusterMatchFunc',
    typing.Callable[[dict], bool],
)

class AttributeMatcher:
    def __init__(self, match_on: str, expected_value: typing.Any):
        self.match_on = match_on
        self.expected_value = expected_value

    def __call__(self, record: typing.Dict[str, typing.Any]) -> bool:
        return (record is not None and self.match_on in record
                and record[self.match_on] == self.expected_value)


def status_matches(status: str) -> ClusterMatchFunc:
    return AttributeMatcher("Status", status)

def wait_until(
        db_cluster_id: str,
        match_fn: ClusterMatchFunc,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a DB cluster with a supplied ID is returned from the RDS API
    and the matching functor returns True.

    Usage:
        from e2e.db_cluster import wait_until, status_matches

        wait_until(
            cluster_id,
            status_matches("available"),
        )

    Raises:
        pytest.fail upon timeout
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)
    
    logging.info(f"Waiting for DB cluster {db_cluster_id} to match condition...")
    
    last_status = None
    attempts = 0
    
    while True:
        attempts += 1
        if datetime.datetime.now() >= timeout:
            pytest.fail(f"Failed to match DBCluster '{db_cluster_id}' before timeout ({timeout_seconds}s). Last status: {last_status}")
            
        try:
            cluster = get(db_cluster_id)
            
            if cluster is not None and 'Status' in cluster:
                current_status = cluster['Status']
                if current_status != last_status:
                    logging.info(f"DB cluster {db_cluster_id} status changed to: {current_status}")
                    last_status = current_status
                    
            if match_fn(cluster):
                logging.info(f"DB cluster {db_cluster_id} matched condition after {attempts} attempts")
                return
                
        except Exception as e:
            logging.warning(f"Error checking DB cluster status (attempt {attempts}): {str(e)}")
            
        # Exponential backoff capped at interval_seconds
        sleep_time = min(interval_seconds, 2 ** (min(attempts, 6)))
        time.sleep(sleep_time)


def wait_until_deleted(
        db_cluster_id: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a DB cluster with a supplied ID is no longer returned from
    the RDS API.

    Usage:
        from e2e.db_cluster import wait_until_deleted

        wait_until_deleted(cluster_id)

    Raises:
        pytest.fail upon timeout or if the DB cluster goes to any other status
        other than 'deleting'
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)
    
    logging.info(f"Waiting for DB cluster {db_cluster_id} to be deleted...")
    
    last_status = None
    attempts = 0
    
    while True:
        attempts += 1
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                f"Timed out waiting for DB cluster '{db_cluster_id}' to be "
                f"deleted in RDS API after {timeout_seconds}s. Last status: {last_status}"
            )
        
        try:
            latest = get(db_cluster_id)
            if latest is None:
                logging.info(f"DB cluster {db_cluster_id} successfully deleted after {attempts} attempts")
                break

            current_status = latest.get('Status', 'unknown')
            if current_status != last_status:
                logging.info(f"DB cluster {db_cluster_id} status changed to: {current_status}")
                last_status = current_status
                
            if current_status != "deleting":
                pytest.fail(
                    f"Status is not 'deleting' for DB cluster '{db_cluster_id}' that was "
                    f"deleted. Status is '{current_status}'"
                )
                
        except Exception as e:
            logging.warning(f"Error checking DB cluster deletion status (attempt {attempts}): {str(e)}")
            
        # Exponential backoff capped at interval_seconds
        sleep_time = min(interval_seconds, 2 ** (min(attempts, 6)))
        time.sleep(sleep_time)


def get(db_cluster_id):
    """Returns a dict containing the DB cluster record from the RDS API.

    If no such DB cluster exists, returns None.
    """
    def _get_cluster(cluster_id):
        c = boto3.client('rds')
        try:
            resp = c.describe_db_clusters(DBClusterIdentifier=cluster_id)
            assert len(resp['DBClusters']) == 1
            return resp['DBClusters'][0]
        except c.exceptions.DBClusterNotFoundFault:
            return None
        except Exception as e:
            logging.warning(f"Error getting DB cluster {cluster_id}: {str(e)}")
            raise

    return retry_on_api_error(_get_cluster, db_cluster_id)


def get_tags(db_cluster_arn):
    """Returns a dict containing the DB cluster's tag records from the RDS API.

    If no such DB cluster exists, returns None.
    """
    def _get_tags(arn):
        c = boto3.client('rds')
        try:
            resp = c.list_tags_for_resource(
                ResourceName=arn,
            )
            return resp['TagList']
        except c.exceptions.DBClusterNotFoundFault:
            return None
        except Exception as e:
            logging.warning(f"Error getting tags for DB cluster {arn}: {str(e)}")
            raise
            
    return retry_on_api_error(_get_tags, db_cluster_arn)
