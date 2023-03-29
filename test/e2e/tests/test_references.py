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

"""Integration tests for resource references"""

import logging
import time

import pytest

from acktest.k8s import condition
from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_rds_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import db_cluster
from e2e import db_cluster_parameter_group
from e2e import db_instance
from e2e import db_parameter_group
from e2e.fixtures import k8s_secret

# Little longer to delete the instance and cluster since it's referred-to from
# the parameter group...
DELETE_INSTANCE_TIMEOUT_SECONDS = 30
DELETE_CLUSTER_TIMEOUT_SECONDS = 60
DELETE_WAIT_AFTER_SECONDS = 10
CREATE_WAIT_AFTER_SECONDS = 10
CHECK_WAIT_AFTER_REF_RESOLVE_SECONDS = 30

# MUP == Master user password...
MUP_NS = "default"
MUP_SEC_CLUSTER_NAME_PREFIX = "dbclustersecrets"
MUP_SEC_INSTANCE_NAME_PREFIX = "dbinstancesecrets"
MUP_SEC_KEY = "master_user_password"
MUP_SEC_VAL = "secretpass123456"


@pytest.fixture(scope="module")
def db_cluster_name():
    return random_suffix_name("ref-db-cluster", 24)


@pytest.fixture(scope="module")
def cpg_name():
    return random_suffix_name("ref-clus-paramgrp", 24)


@pytest.fixture(scope="module")
def pg_name():
    return random_suffix_name("ref-paramgrp", 24)


@pytest.fixture
def ref_db_param_group(pg_name):
    resource_name = pg_name
    replacements = REPLACEMENT_VALUES.copy()
    replacements["DB_PARAMETER_GROUP_NAME"] = resource_name
    replacements["DB_PARAMETER_GROUP_DESC"] = "Aurora PG 14 Params"

    resource_data = load_rds_resource(
        "db_parameter_group_aurora_postgresql14",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, 'dbparametergroups',
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    time.sleep(CREATE_WAIT_AFTER_SECONDS)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield ref, cr, resource_name

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    db_parameter_group.wait_until_deleted(resource_name)


@pytest.fixture
def ref_db_cluster_param_group(cpg_name):
    resource_name = cpg_name
    replacements = REPLACEMENT_VALUES.copy()
    replacements["DB_CLUSTER_PARAMETER_GROUP_NAME"] = resource_name
    replacements["DB_CLUSTER_PARAMETER_GROUP_DESC"] = "Aurora PG 14 Params"

    resource_data = load_rds_resource(
        "db_cluster_parameter_group_aurora_postgresql14",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, 'dbclusterparametergroups',
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    yield ref, cr, resource_name

    # Try to delete, if doesn't already exist
    try:
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted
        time.sleep(DELETE_WAIT_AFTER_SECONDS)
    except:
        pass

    db_cluster_parameter_group.wait_until_deleted(resource_name)


@pytest.fixture(scope="module")
def ref_db_cluster(k8s_secret, db_cluster_name, cpg_name):
    db_name = "mydb"
    secret = k8s_secret(
        MUP_NS,
        random_suffix_name(MUP_SEC_CLUSTER_NAME_PREFIX, 32),
        MUP_SEC_KEY,
        MUP_SEC_VAL,
    )

    replacements = REPLACEMENT_VALUES.copy()
    replacements["DB_CLUSTER_ID"] = db_cluster_name
    replacements["DB_NAME"] = db_name
    replacements["MASTER_USER_PASS_SECRET_NAMESPACE"] = secret.ns
    replacements["MASTER_USER_PASS_SECRET_NAME"] = secret.name
    replacements["MASTER_USER_PASS_SECRET_KEY"] = secret.key
    replacements["DB_CLUSTER_PARAMETER_GROUP_NAME"] = cpg_name

    resource_data = load_rds_resource(
        "db_cluster_ref",
        additional_replacements=replacements,
    )

    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, 'dbclusters',
        db_cluster_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    # NOTE(jaypipes): We specifically do NOT wait for the DBInstance to exist
    # in the RDS API here because we will create the referred-to
    # DBClusterParameterGroup and wait for the reference to be resolved

    yield (ref, cr, db_cluster_name)

    if k8s.get_resource_exists(ref):
        # If all goes properly, we should not hit this because the test cleans
        # up the child resource before exiting...
        _, deleted = k8s.delete_custom_resource(
            ref,
            period_length=DELETE_INSTANCE_TIMEOUT_SECONDS,
        )
        assert deleted

        db_cluster.wait_until_deleted(db_cluster_name)


@pytest.fixture
def ref_db_instance(db_cluster_name, pg_name):
    db_instance_id = random_suffix_name("ref-db-instance", 20)

    replacements = REPLACEMENT_VALUES.copy()
    replacements['COPY_TAGS_TO_SNAPSHOT'] = "False"
    replacements["DB_INSTANCE_ID"] = db_instance_id
    replacements["DB_CLUSTER_ID"] = db_cluster_name
    replacements["DB_PARAMETER_GROUP_NAME"] = pg_name

    resource_data = load_rds_resource(
        "db_instance_ref",
        additional_replacements=replacements,
    )

    # Create the k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, 'dbinstances',
        db_instance_id, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    assert cr is not None
    assert k8s.get_resource_exists(ref)

    # NOTE(jaypipes): We specifically do NOT wait for the DBInstance to exist
    # in the RDS API here because we will create the referred-to
    # DBParameterGroup and wait for the reference to be resolved

    yield (ref, cr, db_instance_id)

    if k8s.get_resource_exists(ref):
        # If all goes properly, we should not hit this because the test cleans
        # up the child resource before exiting...
        _, deleted = k8s.delete_custom_resource(
            ref,
            period_length=DELETE_INSTANCE_TIMEOUT_SECONDS,
        )
        assert deleted

        db_instance.wait_until_deleted(db_instance_id)


@service_marker
@pytest.mark.canary
class TestReferences:
    def test_references(
            self,
            ref_db_cluster,
            ref_db_instance,
            ref_db_param_group,
            ref_db_cluster_param_group,
    ):
        # create the resources in order that initially the reference resolution
        # fails and then when the referenced resource gets created, then all
        # resolutions eventually pass and resources get synced.
        db_cluster_ref, db_cluster_cr, db_cluster_id = ref_db_cluster

        time.sleep(1)

        # The DB Cluster above refers to this Cluster Parameter Group by
        # reference/name
        db_cluster_pg_ref, db_cluster_pg_cr, db_cluster_pg_name = ref_db_cluster_param_group

        time.sleep(CHECK_WAIT_AFTER_REF_RESOLVE_SECONDS)

        db_cluster.wait_until(
            db_cluster_id,
            db_cluster.status_matches("available"),
        )

        time.sleep(60)

        condition.assert_synced(db_cluster_pg_ref)
        condition.assert_synced(db_cluster_ref)

        # Instance refers to parameter group by reference and DB cluster by
        # ID...
        db_instance_ref, db_instance_cr, db_instance_id = ref_db_instance

        # We expect the DB Instance to fail to resolve references because the
        # DB Parameter Group it refers to does not yet exist. Let's create it
        # now.
        db_pg_ref, db_pg_cr, db_pg_name = ref_db_param_group

        time.sleep(CHECK_WAIT_AFTER_REF_RESOLVE_SECONDS)

        db_instance.wait_until(
            db_instance_id,
            db_instance.status_matches("available"),
        )

        time.sleep(60)

        condition.assert_synced(db_pg_ref)
        condition.assert_synced(db_instance_ref)

        # NOTE(jaypipes): We need to manually delete the DB Instance first
        # because pytest fixtures will try to clean up the DB Parameter Group
        # fixture *first* (because it was initialized after DB Instance) but if
        # we try to delete the DB Parameter Group before the DB Instance, the
        # cascading delete protection of resource references will mean the DB
        # Parameter Group won't be deleted.
        _, deleted = k8s.delete_custom_resource(
            db_instance_ref,
            period_length=DELETE_INSTANCE_TIMEOUT_SECONDS,
        )
        assert deleted

        # Wait a bit before trying to delete the cluster since the instance is
        # part of the cluster and sometimes the delete cluster complains if
        # it's too soon after deleting the last DB instance in it.
        time.sleep(60)

        db_instance.wait_until_deleted(db_instance_id)

        # Same for the DB cluster because it refers to the DB cluster
        # parameter group...
        _, deleted = k8s.delete_custom_resource(
            db_cluster_ref,
            period_length=DELETE_CLUSTER_TIMEOUT_SECONDS,
        )
        assert deleted

        db_cluster.wait_until_deleted(db_cluster_id)
