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

"""Integration tests for the RDS API DBCluster resource adoption via
adopt-or-create policy.

1. Create an Aurora PostgreSQL Serverless v2 cluster directly with the RDS
   API, with a database name, I/O-Optimized storage and Database Insights
   advanced mode, leaving every other optional field to its server-side
   default.
2. Adopt it with an adopt-or-create manifest that omits all of those fields.
3. Validate the adopted resource reaches a synced state, the observed values
   are late-initialized into spec, and it stays synced across a reconcile.
"""

import time

import boto3
import pytest

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import condition
from e2e import db_cluster
from e2e import tag

RESOURCE_PLURAL = 'dbclusters'

MAX_WAIT_FOR_SYNCED_MINUTES = 5
MAX_WAIT_FOR_TAG_SECONDS = 120
TAG_CHECK_INTERVAL_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 30


@service_marker
class TestDBClusterAdoption:
    def test_adopt_or_create_late_initializes_defaults(self):
        """Validates that adopting an existing DBCluster with adopt-or-create
        and a sparse spec reaches a synced state instead of reporting a
        nil-vs-default delta on every reconcile, and that the server-side
        defaults are late-initialized into spec.
        """
        db_cluster_id = random_suffix_name("aurora-pg-adopt", 24)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["DB_CLUSTER_ID"] = db_cluster_id

        resource_data = load_rds_resource(
            "db_cluster_aurora_postgres_adopt",
            additional_replacements=replacements,
        )

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            db_cluster_id, namespace="default",
        )

        c = boto3.client('rds')
        try:
            # --- Stage 1: Create the cluster outside of ACK ---
            c.create_db_cluster(
                DBClusterIdentifier=db_cluster_id,
                Engine='aurora-postgresql',
                MasterUsername='dbadmin',
                ManageMasterUserPassword=True,
                ServerlessV2ScalingConfiguration={
                    'MinCapacity': 0,
                    'MaxCapacity': 4,
                },
                # Only reported by DescribeDBClusters when enabled.
                DatabaseName='appdb',
                StorageType='aurora-iopt1',
                DatabaseInsightsMode='advanced',
                EnablePerformanceInsights=True,
                PerformanceInsightsRetentionPeriod=465,
            )
            db_cluster.wait_until(
                db_cluster_id,
                db_cluster.status_matches('available'),
            )

            # --- Stage 2: Adopt it with a sparse manifest ---
            k8s.create_custom_resource(ref, resource_data)
            cr = k8s.wait_resource_consumed_by_controller(ref)

            assert cr is not None
            assert k8s.get_resource_exists(ref)

            # --- Stage 3: Validate adoption reaches synced state ---
            assert k8s.wait_on_condition(
                ref, "ACK.ResourceSynced", "True",
                wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES,
            )

            cr = k8s.get_resource(ref)
            assert cr is not None
            assert cr['metadata']['annotations'].get(
                'services.k8s.aws/adopted'
            ) == 'true'

            # Every observed value omitted from the manifest must be
            # late-initialized into spec, otherwise it shows up as a delta
            # that ModifyDBCluster cannot resolve.
            latest = db_cluster.get(db_cluster_id)
            assert latest is not None
            spec = cr['spec']
            assert spec['autoMinorVersionUpgrade'] == latest['AutoMinorVersionUpgrade']
            assert sorted(spec['availabilityZones']) == sorted(latest['AvailabilityZones'])
            assert spec['backupRetentionPeriod'] == latest['BackupRetentionPeriod']
            assert spec['copyTagsToSnapshot'] == latest['CopyTagsToSnapshot']
            assert spec['databaseInsightsMode'] == latest['DatabaseInsightsMode'] == 'advanced'
            assert spec['databaseName'] == latest['DatabaseName'] == 'appdb'
            assert spec['deletionProtection'] == latest['DeletionProtection']
            assert spec['engineMode'] == latest['EngineMode']
            assert spec['engineVersion'] == latest['EngineVersion']
            assert spec['masterUsername'] == latest['MasterUsername']
            assert spec['networkType'] == latest['NetworkType']
            assert spec['port'] == latest['Port']
            assert spec['preferredBackupWindow'] == latest['PreferredBackupWindow']
            assert spec['preferredMaintenanceWindow'] == latest['PreferredMaintenanceWindow']
            assert spec['storageEncrypted'] == latest['StorageEncrypted']
            assert spec['storageType'] == latest['StorageType'] == 'aurora-iopt1'
            assert spec['performanceInsightsRetentionPeriod'] == \
                latest['PerformanceInsightsRetentionPeriod'] == 465
            assert spec['performanceInsightsKMSKeyID'] == latest['PerformanceInsightsKMSKeyId']
            if 'KmsKeyId' in latest:
                assert spec['kmsKeyID'] == latest['KmsKeyId']
            assert spec['serverlessV2ScalingConfiguration']['secondsUntilAutoPause'] == \
                latest['ServerlessV2ScalingConfiguration']['SecondsUntilAutoPause']

            # A synced resource is not reconciled again until the resync
            # period, so change a tag to force a reconcile and check that it
            # finds no other delta and the resource stays synced.
            k8s.patch_custom_resource(ref, {
                "spec": {"tags": [{"key": "adopt-test", "value": "reconcile"}]},
            })
            expect_tag = {"Key": "adopt-test", "Value": "reconcile"}
            deadline = time.time() + MAX_WAIT_FOR_TAG_SECONDS
            while expect_tag not in tag.clean(db_cluster.get_tags(latest['DBClusterArn'])):
                if time.time() >= deadline:
                    pytest.fail("tag update was not reconciled before timeout")
                time.sleep(TAG_CHECK_INTERVAL_SECONDS)

            assert k8s.wait_on_condition(
                ref, "ACK.ResourceSynced", "True",
                wait_periods=MAX_WAIT_FOR_SYNCED_MINUTES,
            )
            condition.assert_synced(ref)
            latest = db_cluster.get(db_cluster_id)
            assert latest is not None
            assert latest['Status'] == 'available'

        finally:
            # Cleanup: delete K8s resource if it still exists
            try:
                _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            except:
                pass
            time.sleep(DELETE_WAIT_AFTER_SECONDS)

            # Delete the AWS cluster directly since retain policy keeps it
            try:
                c.delete_db_cluster(
                    DBClusterIdentifier=db_cluster_id,
                    SkipFinalSnapshot=True,
                )
                db_cluster.wait_until_deleted(db_cluster_id)
            except c.exceptions.DBClusterNotFoundFault:
                pass
