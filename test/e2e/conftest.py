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

import os
import pytest
import boto3
import logging

from acktest import k8s
from e2e.resource_cleanup import cleanup_old_resources

# Increase default timeouts to handle AWS API latency
def pytest_configure(config):
    # Set longer timeouts for tests
    os.environ['PYTEST_TIMEOUT'] = '900'  # 15 minutes
    
    config.addinivalue_line(
        "markers", "canary: mark test to also run in canary tests"
    )
    config.addinivalue_line(
        "markers", "service(arg): mark test associated with a given service"
    )
    config.addinivalue_line(
        "markers", "slow: mark test as slow to run"
    )
    
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    # Clean up any old test resources that might be lingering
    try:
        logging.info("Running pre-test cleanup of stale AWS resources...")
        cleanup_old_resources()
    except Exception as e:
        logging.warning(f"Error during pre-test resource cleanup: {str(e)}")


def pytest_addoption(parser):
    parser.addoption("--runslow", action="store_true", default=False, help="run slow tests")


def pytest_collection_modifyitems(config, items):
    if config.getoption("--runslow"):
        return
    skip_slow = pytest.mark.skip(reason="need --runslow option to run")
    for item in items:
        if "slow" in item.keywords:
            item.add_marker(skip_slow)

# Provide a k8s client to interact with the integration test cluster
@pytest.fixture(scope='class')
def k8s_client():
    return k8s._get_k8s_api_client()

@pytest.fixture(scope='module')
def rds_client():
    return boto3.client('rds')

@pytest.fixture(scope='module')
def rds_resource():
    return boto3.resource('rds') 

@pytest.fixture(scope='module')
def sts_client():
    return boto3.client('sts')