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

"""Utilities for retrying operations with exponential backoff"""

import time
import logging
import random
import functools
from typing import Callable, TypeVar, Any

import botocore.exceptions

# Default retry settings
DEFAULT_MAX_ATTEMPTS = 5
DEFAULT_BASE_BACKOFF_SECONDS = 2
DEFAULT_MAX_BACKOFF_SECONDS = 60
DEFAULT_JITTER_FACTOR = 0.2

T = TypeVar('T')

def with_retry(
    max_attempts: int = DEFAULT_MAX_ATTEMPTS,
    base_backoff_seconds: float = DEFAULT_BASE_BACKOFF_SECONDS,
    max_backoff_seconds: float = DEFAULT_MAX_BACKOFF_SECONDS, 
    jitter_factor: float = DEFAULT_JITTER_FACTOR,
    retryable_exceptions = (
        botocore.exceptions.ClientError,
        botocore.exceptions.BotoCoreError,
        botocore.exceptions.ConnectionError,
    )
) -> Callable[[Callable[..., T]], Callable[..., T]]:
    """Decorator for retrying functions with exponential backoff
    
    Args:
        max_attempts: Maximum number of retry attempts
        base_backoff_seconds: Initial backoff time in seconds
        max_backoff_seconds: Maximum backoff time in seconds
        jitter_factor: Random jitter factor for backoff timing
        retryable_exceptions: Tuple of exceptions that should trigger a retry
        
    Returns:
        Decorated function that implements retry logic
    """
    def decorator(func: Callable[..., T]) -> Callable[..., T]:
        @functools.wraps(func)
        def wrapper(*args: Any, **kwargs: Any) -> T:
            attempt = 0
            while True:
                try:
                    return func(*args, **kwargs)
                except retryable_exceptions as e:
                    attempt += 1
                    if attempt >= max_attempts:
                        logging.error(f"Max retry attempts ({max_attempts}) reached. Last error: {str(e)}")
                        raise
                    
                    # Calculate backoff with exponential growth and jitter
                    backoff = min(
                        max_backoff_seconds,
                        base_backoff_seconds * (2 ** (attempt - 1))
                    )
                    
                    # Add jitter to avoid thundering herd problems
                    jitter = backoff * jitter_factor * random.random()
                    sleep_time = backoff + jitter
                    
                    logging.warning(
                        f"Retrying after error: {str(e)}. "
                        f"Sleeping for {sleep_time:.2f}s. "
                        f"Attempt {attempt}/{max_attempts-1}"
                    )
                    time.sleep(sleep_time)
        
        return wrapper
    
    return decorator


def retry_on_api_error(func, *args, **kwargs):
    """Helper function to retry AWS API calls with exponential backoff"""
    @with_retry()
    def wrapped():
        return func(*args, **kwargs)
    
    return wrapped()

def wait_for_resources_deleted(resource_type: str, name_pattern: str, timeout_seconds: int = 300):
    """Wait for all resources of a certain type matching a pattern to be deleted.
    
    This helps prevent test failures due to background cleanup still in progress.
    
    Args:
        resource_type: Type of resource (e.g., "db_instance")
        name_pattern: Regex pattern to match resource names
        timeout_seconds: Maximum time to wait for resources to be deleted
    """
    import re
    import time
    from datetime import datetime, timedelta
    import logging
    import boto3
    
    logging.info(f"Waiting for {resource_type} resources matching '{name_pattern}' to be deleted...")
    
    rds_client = boto3.client('rds')
    pattern = re.compile(name_pattern)
    now = datetime.now()
    timeout = now + timedelta(seconds=timeout_seconds)
    
    while datetime.now() < timeout:
        resources_found = 0
        
        try:
            if resource_type == "db_instance":
                paginator = rds_client.get_paginator('describe_db_instances')
                for page in paginator.paginate():
                    for instance in page.get('DBInstances', []):
                        if pattern.match(instance['DBInstanceIdentifier']):
                            resources_found += 1
                            status = instance.get('DBInstanceStatus', 'unknown')
                            logging.info(f"DB instance {instance['DBInstanceIdentifier']} still exists with status: {status}")
                            
            elif resource_type == "db_cluster":
                paginator = rds_client.get_paginator('describe_db_clusters')
                for page in paginator.paginate():
                    for cluster in page.get('DBClusters', []):
                        if pattern.match(cluster['DBClusterIdentifier']):
                            resources_found += 1
                            status = cluster.get('Status', 'unknown')
                            logging.info(f"DB cluster {cluster['DBClusterIdentifier']} still exists with status: {status}")
            
            if resources_found == 0:
                logging.info(f"All {resource_type} resources matching '{name_pattern}' have been deleted")
                return True
                
            logging.info(f"Found {resources_found} {resource_type}(s) still being deleted, waiting...")
            time.sleep(15)  # Check every 15 seconds
            
        except Exception as e:
            logging.warning(f"Error checking for {resource_type} deletion: {str(e)}")
            time.sleep(5)
    
    logging.warning(f"Timed out waiting for {resource_type} resources to be deleted")
    return False 