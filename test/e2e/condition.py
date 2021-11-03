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

"""Utility functions to help processing Kubernetes resource conditions"""

# TODO(jaypipes): Move these functions to acktest library. The reason these are
# here is because the existing k8s.assert_condition_state_message doesn't
# actually assert anything. It returns true or false and logs messages.

import pytest

from acktest.k8s import resource as k8s

CONDITION_TYPE_ADOPTED = "ACK.Adopted"
CONDITION_TYPE_RESOURCE_SYNCED = "ACK.ResourceSynced"
CONDITION_TYPE_TERMINAL = "ACK.Terminal"
CONDITION_TYPE_RECOVERABLE = "ACK.Recoverable"
CONDITION_TYPE_ADVISORY = "ACK.Advisory"
CONDITION_TYPE_LATE_INITIALIZED = "ACK.LateInitialized"


def assert_type_status(
    ref: k8s.CustomResourceReference,
    cond_type_match: str = CONDITION_TYPE_RESOURCE_SYNCED,
    cond_status_match: bool = True,
):
    """Asserts that the supplied resource has a condition of type
    ACK.ResourceSynced and that the Status of this condition is True.

    Usage:
        from acktest.k8s import resource as k8s

        from e2e import condition

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            db_cluster_id, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        k8s.wait_resource_consumed_by_controller(ref)
        condition.assert_type_status(
            ref,
            condition.CONDITION_TYPE_RESOURCE_SYNCED,
            False)

    Raises:
        pytest.fail when condition of the specified type is not found or is not
        in the supplied status.
    """
    cond = k8s.get_resource_condition(ref, cond_type_match)
    if cond is None:
        msg = (f"Failed to find {cond_type_match} condition in "
               f"resource {ref}")
        pytest.fail(msg)

    cond_status = cond.get('status', None)
    if str(cond_status) != str(cond_status_match):
        msg = (f"Expected {cond_type_match} condition to "
               f"have status {cond_status_match} but found {cond_status}")
        pytest.fail(msg)


def assert_synced_status(
    ref: k8s.CustomResourceReference,
    cond_status_match: bool,
):
    """Asserts that the supplied resource has a condition of type
    ACK.ResourceSynced and that the Status of this condition is True.

    Usage:
        from acktest.k8s import resource as k8s

        from e2e import condition

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            db_cluster_id, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        k8s.wait_resource_consumed_by_controller(ref)
        condition.assert_synced_status(ref, False)

    Raises:
        pytest.fail when ACK.ResourceSynced condition is not found or is not in
        a True status.
    """
    assert_type_status(ref, CONDITION_TYPE_RESOURCE_SYNCED, cond_status_match)


def assert_synced(ref: k8s.CustomResourceReference):
    """Asserts that the supplied resource has a condition of type
    ACK.ResourceSynced and that the Status of this condition is True.

    Usage:
        from acktest.k8s import resource as k8s

        from e2e import condition

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            db_cluster_id, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        k8s.wait_resource_consumed_by_controller(ref)
        condition.assert_synced(ref)

    Raises:
        pytest.fail when ACK.ResourceSynced condition is not found or is not in
        a True status.
    """
    return assert_synced_status(ref, True)


def assert_not_synced(ref: k8s.CustomResourceReference):
    """Asserts that the supplied resource has a condition of type
    ACK.ResourceSynced and that the Status of this condition is False.

    Usage:
        from acktest.k8s import resource as k8s

        from e2e import condition

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            db_cluster_id, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        k8s.wait_resource_consumed_by_controller(ref)
        condition.assert_not_synced(ref)

    Raises:
        pytest.fail when ACK.ResourceSynced condition is not found or is not in
        a False status.
    """
    return assert_synced_status(ref, False)
