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