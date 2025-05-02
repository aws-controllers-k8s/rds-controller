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

"""Utilities for working with DB instance resources"""

import datetime
import time
import typing
import logging
import botocore.exceptions

import boto3
import pytest
from e2e.retry_util import retry_on_api_error

DEFAULT_WAIT_UNTIL_TIMEOUT_SECONDS = 60*40
DEFAULT_WAIT_UNTIL_INTERVAL_SECONDS = 15
DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 60*30
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15

InstanceMatchFunc = typing.NewType(
    'InstanceMatchFunc',
    typing.Callable[[dict], bool],
)

class StatusMatcher:
    def __init__(self, status):
        self.match_on = status

    def __call__(self, record: dict) -> bool:
        return (record is not None and 'DBInstanceStatus' in record
                and record['DBInstanceStatus'] == self.match_on)


def status_matches(status: str) -> InstanceMatchFunc:
    return StatusMatcher(status)


def wait_until(
        db_instance_id: str,
        match_fn: InstanceMatchFunc,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a DB instance with a supplied ID is returned from the RDS API
    and the matching functor returns True.

    Usage:
        from e2e.db_instance import wait_until, status_matches

        wait_until(
            instance_id,
            status_matches("available"),
        )

    Raises:
        pytest.fail upon timeout
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)
    
    logging.info(f"Waiting for DB instance {db_instance_id} to match condition...")
    
    last_status = None
    attempts = 0
    
    while True:
        attempts += 1
        if datetime.datetime.now() >= timeout:
            pytest.fail(f"Failed to match DBInstance '{db_instance_id}' before timeout ({timeout_seconds}s). Last status: {last_status}")
            
        try:
            instance = get(db_instance_id)
            
            if instance is not None and 'DBInstanceStatus' in instance:
                current_status = instance['DBInstanceStatus']
                if current_status != last_status:
                    logging.info(f"DB instance {db_instance_id} status changed to: {current_status}")
                    last_status = current_status
                    
            if match_fn(instance):
                logging.info(f"DB instance {db_instance_id} matched condition after {attempts} attempts")
                return
                
        except Exception as e:
            logging.warning(f"Error checking DB instance status (attempt {attempts}): {str(e)}")
            
        # Exponential backoff capped at interval_seconds
        sleep_time = min(interval_seconds, 2 ** (min(attempts, 6)))
        time.sleep(sleep_time)


def wait_until_deleted(
        db_instance_id: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a DB instance with a supplied ID is no longer returned from
    the RDS API.

    Usage:
        from e2e.db_instance import wait_until_deleted

        wait_until_deleted(instance_id)

    Raises:
        pytest.fail upon timeout or if the DB instance goes to any other status
        other than 'deleting'
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)
    
    logging.info(f"Waiting for DB instance {db_instance_id} to be deleted...")
    
    last_status = None
    attempts = 0
    
    while True:
        attempts += 1
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                f"Timed out waiting for DB instance '{db_instance_id}' to be "
                f"deleted in RDS API after {timeout_seconds}s. Last status: {last_status}"
            )
            
        try:
            latest = get(db_instance_id)
            if latest is None:
                logging.info(f"DB instance {db_instance_id} successfully deleted after {attempts} attempts")
                break
    
            current_status = latest.get('DBInstanceStatus', 'unknown')
            if current_status != last_status:
                logging.info(f"DB instance {db_instance_id} status changed to: {current_status}")
                last_status = current_status
                
            if current_status != "deleting":
                pytest.fail(
                    f"Status is not 'deleting' for DB instance '{db_instance_id}' that was "
                    f"deleted. Status is '{current_status}'"
                )
                
        except Exception as e:
            logging.warning(f"Error checking DB instance deletion status (attempt {attempts}): {str(e)}")
            
        # Exponential backoff capped at interval_seconds
        sleep_time = min(interval_seconds, 2 ** (min(attempts, 6)))
        time.sleep(sleep_time)


def get(db_instance_id):
    """Returns a dict containing the DB instance record from the RDS API.

    If no such DB instance exists, returns None.
    """
    def _get_instance(instance_id):
        c = boto3.client('rds')
        try:
            resp = c.describe_db_instances(DBInstanceIdentifier=instance_id)
            assert len(resp['DBInstances']) == 1
            return resp['DBInstances'][0]
        except c.exceptions.DBInstanceNotFoundFault:
            return None
        except Exception as e:
            logging.warning(f"Error getting DB instance {instance_id}: {str(e)}")
            raise
    
    return retry_on_api_error(_get_instance, db_instance_id)


def get_tags(db_instance_arn):
    """Returns a dict containing the DB instance's tag records from the RDS API.

    If no such DB instance exists, returns None.
    """
    def _get_tags(arn):
        c = boto3.client('rds')
        try:
            resp = c.list_tags_for_resource(
                ResourceName=arn,
            )
            return resp['TagList']
        except c.exceptions.DBInstanceNotFoundFault:
            return None
        except Exception as e:
            logging.warning(f"Error getting tags for DB instance {arn}: {str(e)}")
            raise
            
    return retry_on_api_error(_get_tags, db_instance_arn)
