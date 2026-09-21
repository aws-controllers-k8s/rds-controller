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

import boto3
import pytest

DEFAULT_WAIT_UNTIL_TIMEOUT_SECONDS = 60*30
DEFAULT_WAIT_UNTIL_INTERVAL_SECONDS = 15
DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 60*20
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15

# RDS emits this message when it applies a master password modification, as
# RDS-EVENT-0016 in the "configuration change" category. DescribeEvents returns
# only the message, not the RDS-EVENT-nnnn identifier, so the message is what
# can be matched on. Compared case-insensitively as a substring.
# https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_Events.Messages.html
MASTER_CREDENTIALS_RESET_MESSAGE = "reset master credentials"

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

    while not match_fn(get(db_instance_id)):
        if datetime.datetime.now() >= timeout:
            pytest.fail("failed to match DBInstance before timeout")
        time.sleep(interval_seconds)


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

    while True:
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                "Timed out waiting for DB instance to be "
                "deleted in RDS API"
            )
        time.sleep(interval_seconds)

        latest = get(db_instance_id)
        if latest is None:
            break

        if latest['DBInstanceStatus'] != "deleting":
            pytest.fail(
                "Status is not 'deleting' for DB instance that was "
                "deleted. Status is " + latest['DBInstanceStatus']
            )


def get(db_instance_id):
    """Returns a dict containing the DB instance record from the RDS API.

    If no such DB instance exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.describe_db_instances(DBInstanceIdentifier=db_instance_id)
        assert len(resp['DBInstances']) == 1
        return resp['DBInstances'][0]
    except c.exceptions.DBInstanceNotFoundFault:
        return None


def wait_for_master_credentials_reset(
        db_instance_id: str,
        since: datetime.datetime,
        timeout_seconds: int = 60*5,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_INTERVAL_SECONDS,
    ) -> None:
    """Waits until RDS reports having reset the DB instance's master credentials.

    Verifies on the AWS side that a master password modification was actually
    applied, rather than inferring it from controller state. RDS emits
    "Reset master credentials." (RDS-EVENT-0016, category "configuration
    change") when it applies the change.

    The event is used in preference to DBInstanceStatus or
    PendingModifiedValues.MasterUserPassword because both of those are
    transient, whereas events are retained (14 days by default) and so can be
    polled for without racing the modification window.

    `since` bounds the query so that an earlier reset on the same instance
    cannot satisfy it; pass a timestamp taken before requesting the change.

    Usage:
        from e2e import db_instance
        before = datetime.datetime.now(datetime.timezone.utc)
        # ... request the password change ...
        db_instance.wait_for_master_credentials_reset(instance_id, before)

    Raises:
        pytest.fail upon timeout
    """
    c = boto3.client('rds')
    deadline = time.time() + timeout_seconds
    seen = []
    while time.time() < deadline:
        resp = c.describe_events(
            SourceIdentifier=db_instance_id,
            SourceType='db-instance',
            EventCategories=['configuration change'],
            StartTime=since,
        )
        events = resp.get('Events', [])
        seen = [e.get('Message') for e in events]
        if any(MASTER_CREDENTIALS_RESET_MESSAGE in (e.get('Message') or '').lower()
               for e in events):
            return
        time.sleep(interval_seconds)
    pytest.fail(
        f"timed out after {timeout_seconds}s waiting for RDS to report "
        f"{MASTER_CREDENTIALS_RESET_MESSAGE!r} on {db_instance_id}; "
        f"configuration-change events since {since}: {seen}"
    )


def get_tags(db_instance_arn):
    """Returns a dict containing the DB instance's tag records from the RDS API.

    If no such DB instance exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.list_tags_for_resource(
            ResourceName=db_instance_arn,
        )
        return resp['TagList']
    except c.exceptions.DBInstanceNotFoundFault:
        return None
