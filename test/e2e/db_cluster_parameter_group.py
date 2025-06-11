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
import logging
import botocore.exceptions
from e2e.retry_util import retry_on_api_error

import boto3
import pytest
from e2e import CRD_GROUP, CRD_VERSION

DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS = 60*10
DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS = 15


def wait_until_deleted(
        db_cluster_parameter_group_name: str,
        timeout_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_TIMEOUT_SECONDS,
        interval_seconds: int = DEFAULT_WAIT_UNTIL_DELETED_INTERVAL_SECONDS,
    ) -> None:
    """Waits until a DB cluster parameter group with a supplied name is no longer
    returned from the RDS API.

    Usage:
        from e2e.db_cluster_parameter_group import wait_until_deleted

        wait_until_deleted(db_cluster_parameter_group_name)

    Raises:
        pytest.fail upon timeout
    """
    now = datetime.datetime.now()
    timeout = now + datetime.timedelta(seconds=timeout_seconds)
    
    logging.info(f"Waiting for DB cluster parameter group {db_cluster_parameter_group_name} to be deleted...")
    
    attempts = 0
    
    while True:
        attempts += 1
        if datetime.datetime.now() >= timeout:
            pytest.fail(
                f"Timed out waiting for DB cluster parameter group '{db_cluster_parameter_group_name}' to be "
                f"deleted in RDS API after {timeout_seconds}s"
            )
            
        try:
            latest = get(db_cluster_parameter_group_name)
            if latest is None:
                logging.info(f"DB cluster parameter group {db_cluster_parameter_group_name} successfully deleted after {attempts} attempts")
                break
                
            logging.info(f"DB cluster parameter group {db_cluster_parameter_group_name} still exists, waiting...")
            
        except Exception as e:
            logging.warning(f"Error checking DB cluster parameter group deletion status (attempt {attempts}): {str(e)}")
            
        # Exponential backoff capped at interval_seconds
        sleep_time = min(interval_seconds, 2 ** (min(attempts, 6)))
        time.sleep(sleep_time)


def get(db_cluster_parameter_group_name):
    """Returns a dict containing the DB cluster parameter group from the RDS API.

    If no such parameter group exists, returns None.
    """
    def _get_cluster_parameter_group(pg_name):
        c = boto3.client('rds')
        try:
            resp = c.describe_db_cluster_parameter_groups(
                DBClusterParameterGroupName=pg_name
            )
            assert len(resp['DBClusterParameterGroups']) == 1
            return resp['DBClusterParameterGroups'][0]
        except c.exceptions.DBParameterGroupNotFoundFault:
            return None
        except Exception as e:
            logging.warning(f"Error getting DB cluster parameter group {pg_name}: {str(e)}")
            raise
    
    return retry_on_api_error(_get_cluster_parameter_group, db_cluster_parameter_group_name)

def get_parameters(db_cluster_parameter_group_name):
    """Returns a dict containing the paramters of a given parameter group

    If no such DB cluster parameter group exists, returns None.
    """
    c = boto3.client('rds')
    try:
        resp = c.describe_db_cluster_parameters(
            DBClusterParameterGroupName=db_cluster_parameter_group_name,
        )
        return resp['Parameters']
    except c.exceptions.DBClusterParameterGroupNotFoundFault:
        return None

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

def ensure_resource_reference(ref_or_dict, resource_name=None):
    """Ensures we have a proper CustomResourceReference object.
    
    If ref_or_dict is already a CustomResourceReference, returns it.
    If ref_or_dict is a dict, creates a CustomResourceReference from it.
    
    Args:
        ref_or_dict: Either a CustomResourceReference or a dict
        resource_name: Optional resource name to use if not in ref_or_dict
        
    Returns:
        A CustomResourceReference object
    """
    from acktest.k8s import resource as k8s
    
    if hasattr(ref_or_dict, 'namespace'):
        # Already a CustomResourceReference
        return ref_or_dict
        
    # It's a dict, create a CustomResourceReference
    name = resource_name
    if not name and isinstance(ref_or_dict, dict):
        # Try to extract name from metadata
        if 'metadata' in ref_or_dict and 'name' in ref_or_dict['metadata']:
            name = ref_or_dict['metadata']['name']
            
    if not name:
        # Fallback or error case
        logging.warning("Could not determine resource name for CustomResourceReference")
        return ref_or_dict
            
    return k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, 'dbclusterparametergroups',
        name, namespace="default"
    )
