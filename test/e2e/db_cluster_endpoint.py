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

"""Utilities for working with DB cluster endpoint resources"""

import time
from typing import Dict, Optional, List

import boto3
import pytest

def get_client():
    return boto3.client('rds')

def get(db_cluster_endpoint_identifier: str) -> Optional[Dict]:
    c = get_client()
    try:
        resp = c.describe_db_cluster_endpoints(
            DBClusterEndpointIdentifier=db_cluster_endpoint_identifier,
        )
        if len(resp['DBClusterEndpoints']) == 0:
            return None
        return resp['DBClusterEndpoints'][0]
    except c.exceptions.DBClusterEndpointNotFoundFault:
        return None

def get_tags(endpoint_arn: str) -> List[Dict]:
    c = get_client()
    try:
        resp = c.list_tags_for_resource(
            ResourceName=endpoint_arn,
        )
        return resp['TagList']
    except c.exceptions.DBClusterEndpointNotFoundFault:
        return []

def wait_until_deleted(db_cluster_endpoint_identifier: str, max_attempts: int = 60) -> None:
    attempt = 0
    while attempt < max_attempts:
        if get(db_cluster_endpoint_identifier) is None:
            return
        attempt += 1
        time.sleep(10)
    pytest.fail(
        "Timed out waiting for DB snapshot to be "
        "deleted in RDS API"
    )