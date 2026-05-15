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

"""Integration tests for the RDS API DBInstance resource adoption via
adopt-or-create policy.

This test uses the same adopt-or-create manifest for both stages:
1. First apply — no existing AWS resource, so adopt-or-create creates it.
2. Delete the K8s CR with retain policy — AWS instance stays alive.
3. Second apply — AWS resource exists, so adopt-or-create adopts it.
4. Validate the adopted resource reaches a synced state and spec fields match.
"""

import time

import boto3
import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import condition
from e2e import db_instance

RESOURCE_PLURAL = 'dbinstances'

MAX_WAIT_FOR_SYNCED_MINUTES = 20
DELETE_WAIT_AFTER_SECONDS = 30


@service_marker
class TestDBInstanceAdoption:
    def test_adopt_or_create_reaches_synced(self):
        """Validates that when adopting an existing DBInstance with
        adopt-or-create, the resource reaches a synced state and the spec
        fields explicitly set in the manifest match the AWS resource.

        Uses the same manifest for both initial creation and subsequent
        adoption to ensure spec parity.
        """
        db_instance_id = random_suffix_name("pg17-adopt", 24)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["DB_INSTANCE_ID"] = db_instance_id

        resource_data = load_rds_resource(
            "db_instance_postgres17_adopt",
            additional_replacements=replacements,
        )

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            db_instance_id, namespace="default",
        )

        try:
            # --- Stage 1: Initial create via adopt-or-create ---
            k8s.create_custom_resource(ref, resource_data)
            cr = k8s.wait_resource_consumed_by_controller(ref)

            assert cr is not None
            assert k8s.get_resource_exists(ref)

            # Wait for the resource to reach synced state (instance available)
            assert k8s.wait_on_condition(
                ref, "ACK.ResourceSynced", "True",
                wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES,
            )

            latest = db_instance.get(db_instance_id)
            assert latest is not None
            assert latest['DBInstanceStatus'] == 'available'

            # --- Stage 2: Delete K8s CR, retain AWS resource ---
            _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            assert deleted
            time.sleep(DELETE_WAIT_AFTER_SECONDS)

            # Confirm K8s resource is gone but AWS instance persists
            assert not k8s.get_resource_exists(ref)
            latest = db_instance.get(db_instance_id)
            assert latest is not None
            assert latest['DBInstanceStatus'] == 'available'

            # --- Stage 3: Re-apply same manifest to adopt existing instance ---
            k8s.create_custom_resource(ref, resource_data)
            cr = k8s.wait_resource_consumed_by_controller(ref)

            assert cr is not None
            assert k8s.get_resource_exists(ref)

            # --- Stage 4: Validate adoption reaches synced state ---
            assert k8s.wait_on_condition(
                ref, "ACK.ResourceSynced", "True",
                wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES,
            )

            cr = k8s.get_resource(ref)
            assert cr is not None
            assert 'status' in cr
            assert 'dbInstanceStatus' in cr['status']
            assert cr['status']['dbInstanceStatus'] == 'available'
            condition.assert_synced(ref)

            # Verify that spec fields explicitly set in the manifest match AWS
            latest = db_instance.get(db_instance_id)
            assert latest is not None
            assert latest['DBInstanceStatus'] == 'available'
            assert latest['AllocatedStorage'] == 20
            assert latest['DBInstanceClass'] == 'db.t4g.micro'
            assert latest['Engine'] == 'postgres'
            assert latest['EngineVersion'].startswith('17')
            assert 'MasterUserSecret' in latest
            assert latest['MasterUsername'] == 'dbadmin'
            assert latest['MultiAZ'] is False
            assert latest['StorageType'] == 'gp2'

            # Verify annotations are preserved
            assert cr['metadata']['annotations'].get(
                'services.k8s.aws/adoption-policy'
            ) == 'adopt-or-create'
            assert cr['metadata']['annotations'].get(
                'services.k8s.aws/deletion-policy'
            ) == 'retain'

        finally:
            # Cleanup: delete K8s resource if it still exists
            try:
                _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            except:
                pass
            time.sleep(DELETE_WAIT_AFTER_SECONDS)

            # Delete the AWS instance directly since retain policy keeps it
            c = boto3.client('rds')
            try:
                c.delete_db_instance(
                    DBInstanceIdentifier=db_instance_id,
                    SkipFinalSnapshot=True,
                    DeleteAutomatedBackups=True,
                )
                db_instance.wait_until_deleted(db_instance_id)
            except c.exceptions.DBInstanceNotFoundFault:
                pass
