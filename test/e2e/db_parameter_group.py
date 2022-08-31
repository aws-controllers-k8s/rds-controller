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

import boto3


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
