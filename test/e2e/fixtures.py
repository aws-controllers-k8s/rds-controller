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

"""Fixtures common to all RDS controller tests"""

import dataclasses

from acktest.k8s import resource as k8s
import boto3
import pytest


@pytest.fixture(scope="module")
def rds_client():
    return boto3.client('rds')


@dataclasses.dataclass
class SecretKeyReference:
    ns: str
    name: str
    key: str
    val: str


@pytest.fixture(scope="module")
def k8s_secret():
    """Manages the lifecycle of a Kubernetes Secret for use in tests.

    Usage:
        from e2e.fixtures import k8s_secret

        class TestThing:
            def test_thing(self, k8s_secret):
                secret = k8s_secret(
                    "default", "mysecret", "mykey", "myval",
                )
    """
    created = []
    def _k8s_secret(ns, name, key, val):
        k8s.create_opaque_secret(ns, name, key, val)
        secret_ref = SecretKeyReference(ns, name, key, val)
        created.append(secret_ref)
        return secret_ref

    yield _k8s_secret

    for secret_ref in created:
        k8s.delete_secret(secret_ref.ns, secret_ref.name)
