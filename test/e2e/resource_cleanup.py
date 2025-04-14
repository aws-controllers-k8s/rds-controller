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

"""Utility for cleaning up AWS RDS resources that could be left over from failed tests."""

import logging
import re
import boto3
import time
from datetime import datetime, timedelta
from typing import List, Dict, Any, Set

# Resource patterns to match test resources
RESOURCE_PATTERNS = {
    "db_instance": r"^ref-db-instance-|^pg14-t3-micro-",
    "db_cluster": r"^ref-db-cluster-",
    "db_cluster_parameter_group": r"^ref-clus-paramgrp-",
    "db_parameter_group": r"^ref-paramgrp-",
    "db_snapshot": r"^ref-snapshot-",
    "db_cluster_snapshot": r"^ref-cluster-snapshot-",
    "global_cluster": r"^ref-global-cluster-",
}

# Maximum age of resources to clean up (in hours)
MAX_RESOURCE_AGE_HOURS = 24

def get_old_resources(client: Any, resource_type: str, name_pattern: str) -> List[Dict[str, Any]]:
    """Get resources matching pattern and older than MAX_RESOURCE_AGE_HOURS."""
    try:
        resources = []
        pattern = re.compile(name_pattern)
        cutoff_time = datetime.now() - timedelta(hours=MAX_RESOURCE_AGE_HOURS)
        
        if resource_type == "db_instance":
            paginator = client.get_paginator('describe_db_instances')
            for page in paginator.paginate():
                for instance in page['DBInstances']:
                    if pattern.match(instance['DBInstanceIdentifier']):
                        if instance.get('InstanceCreateTime', datetime.now()) < cutoff_time:
                            resources.append(instance)
                            
        elif resource_type == "db_cluster":
            paginator = client.get_paginator('describe_db_clusters')
            for page in paginator.paginate():
                for cluster in page['DBClusters']:
                    if pattern.match(cluster['DBClusterIdentifier']):
                        if cluster.get('ClusterCreateTime', datetime.now()) < cutoff_time:
                            resources.append(cluster)
                            
        elif resource_type == "db_parameter_group":
            paginator = client.get_paginator('describe_db_parameter_groups')
            for page in paginator.paginate():
                for pg in page['DBParameterGroups']:
                    if pattern.match(pg['DBParameterGroupName']):
                        resources.append(pg)
                            
        elif resource_type == "db_cluster_parameter_group":
            paginator = client.get_paginator('describe_db_cluster_parameter_groups')
            for page in paginator.paginate():
                for cpg in page['DBClusterParameterGroups']:
                    if pattern.match(cpg['DBClusterParameterGroupName']):
                        resources.append(cpg)
                            
        elif resource_type == "db_snapshot":
            paginator = client.get_paginator('describe_db_snapshots')
            for page in paginator.paginate():
                for snapshot in page['DBSnapshots']:
                    if pattern.match(snapshot['DBSnapshotIdentifier']):
                        if snapshot.get('SnapshotCreateTime', datetime.now()) < cutoff_time:
                            resources.append(snapshot)
                            
        elif resource_type == "db_cluster_snapshot":
            paginator = client.get_paginator('describe_db_cluster_snapshots')
            for page in paginator.paginate():
                for snapshot in page['DBClusterSnapshots']:
                    if pattern.match(snapshot['DBClusterSnapshotIdentifier']):
                        if snapshot.get('SnapshotCreateTime', datetime.now()) < cutoff_time:
                            resources.append(snapshot)
                            
        elif resource_type == "global_cluster":
            response = client.describe_global_clusters()
            for cluster in response.get('GlobalClusters', []):
                if pattern.match(cluster['GlobalClusterIdentifier']):
                    resources.append(cluster)
                    
        return resources
        
    except Exception as e:
        logging.warning(f"Error listing {resource_type} resources: {str(e)}")
        return []

def delete_resource(client: Any, resource_type: str, resource: Dict[str, Any]) -> bool:
    """Delete a specific resource."""
    try:
        if resource_type == "db_instance":
            client.delete_db_instance(
                DBInstanceIdentifier=resource['DBInstanceIdentifier'],
                SkipFinalSnapshot=True,
                DeleteAutomatedBackups=True
            )
            logging.info(f"Submitted delete for DB instance: {resource['DBInstanceIdentifier']}")
            
        elif resource_type == "db_cluster":
            client.delete_db_cluster(
                DBClusterIdentifier=resource['DBClusterIdentifier'],
                SkipFinalSnapshot=True
            )
            logging.info(f"Submitted delete for DB cluster: {resource['DBClusterIdentifier']}")
            
        elif resource_type == "db_parameter_group":
            client.delete_db_parameter_group(
                DBParameterGroupName=resource['DBParameterGroupName']
            )
            logging.info(f"Deleted DB parameter group: {resource['DBParameterGroupName']}")
            
        elif resource_type == "db_cluster_parameter_group":
            client.delete_db_cluster_parameter_group(
                DBClusterParameterGroupName=resource['DBClusterParameterGroupName']
            )
            logging.info(f"Deleted DB cluster parameter group: {resource['DBClusterParameterGroupName']}")
            
        elif resource_type == "db_snapshot":
            client.delete_db_snapshot(
                DBSnapshotIdentifier=resource['DBSnapshotIdentifier']
            )
            logging.info(f"Deleted DB snapshot: {resource['DBSnapshotIdentifier']}")
            
        elif resource_type == "db_cluster_snapshot":
            client.delete_db_cluster_snapshot(
                DBClusterSnapshotIdentifier=resource['DBClusterSnapshotIdentifier']
            )
            logging.info(f"Deleted DB cluster snapshot: {resource['DBClusterSnapshotIdentifier']}")
            
        elif resource_type == "global_cluster":
            client.delete_global_cluster(
                GlobalClusterIdentifier=resource['GlobalClusterIdentifier']
            )
            logging.info(f"Deleted global cluster: {resource['GlobalClusterIdentifier']}")
            
        return True
        
    except Exception as e:
        logging.warning(f"Error deleting {resource_type}: {str(e)}")
        return False

def cleanup_old_resources():
    """Find and clean up old test resources."""
    rds_client = boto3.client('rds')
    
    # Delete resources in order (instances first, then clusters, etc.)
    resource_order = [
        "db_instance", 
        "db_cluster", 
        "db_snapshot", 
        "db_cluster_snapshot", 
        "global_cluster",
        "db_parameter_group", 
        "db_cluster_parameter_group"
    ]
    
    deleted_count = 0
    
    for resource_type in resource_order:
        pattern = RESOURCE_PATTERNS.get(resource_type)
        if not pattern:
            continue
            
        resources = get_old_resources(rds_client, resource_type, pattern)
        if not resources:
            logging.info(f"No old {resource_type} resources found to clean up")
            continue
            
        logging.info(f"Found {len(resources)} old {resource_type} resources to clean up")
        
        for resource in resources:
            if delete_resource(rds_client, resource_type, resource):
                deleted_count += 1
                
        # Wait a bit between resource types to allow AWS to process deletions
        if resources:
            time.sleep(5)
            
    logging.info(f"Cleanup complete. Submitted deletion for {deleted_count} resources.")
    
    return deleted_count

def force_cleanup_test_resources():
    """Force cleanup any resources that might have been created during tests.
    
    This is more aggressive than the cleanup_old_resources function and is 
    intended to be called at the end of a test run to ensure all test
    resources are properly removed.
    """
    logging.info("Performing forced cleanup of test resources...")
    
    # Clean up in the right order to avoid dependency issues
    cleanup_order = [
        "db_instance",
        "db_cluster", 
        "db_snapshot",
        "db_cluster_snapshot",
        "global_cluster",
        "db_parameter_group",
        "db_cluster_parameter_group"
    ]
    
    deleted_count = 0
    rds_client = boto3.client('rds')
    
    for resource_type in cleanup_order:
        pattern = RESOURCE_PATTERNS.get(resource_type)
        logging.info(f"Forcibly cleaning up {resource_type} resources...")
        
        try:
            # Find all resources matching the patterns, regardless of age
            resources = []
            pattern_regex = re.compile(pattern)
            
            if resource_type == "db_instance":
                try:
                    response = rds_client.describe_db_instances()
                    for instance in response.get('DBInstances', []):
                        if pattern_regex.match(instance['DBInstanceIdentifier']):
                            logging.info(f"Found test DB instance: {instance['DBInstanceIdentifier']}")
                            resources.append(instance)
                except Exception as e:
                    logging.warning(f"Error listing {resource_type}: {str(e)}")
                
            elif resource_type == "db_cluster":
                try:
                    response = rds_client.describe_db_clusters()
                    for cluster in response.get('DBClusters', []):
                        if pattern_regex.match(cluster['DBClusterIdentifier']):
                            logging.info(f"Found test DB cluster: {cluster['DBClusterIdentifier']}")
                            resources.append(cluster)
                except Exception as e:
                    logging.warning(f"Error listing {resource_type}: {str(e)}")
                
            elif resource_type == "db_parameter_group":
                try:
                    response = rds_client.describe_db_parameter_groups()
                    for pg in response.get('DBParameterGroups', []):
                        if pattern_regex.match(pg['DBParameterGroupName']):
                            logging.info(f"Found test DB parameter group: {pg['DBParameterGroupName']}")
                            resources.append(pg)
                except Exception as e:
                    logging.warning(f"Error listing {resource_type}: {str(e)}")
                
            elif resource_type == "db_cluster_parameter_group":
                try:
                    response = rds_client.describe_db_cluster_parameter_groups()
                    for cpg in response.get('DBClusterParameterGroups', []):
                        if pattern_regex.match(cpg['DBClusterParameterGroupName']):
                            logging.info(f"Found test DB cluster parameter group: {cpg['DBClusterParameterGroupName']}")
                            resources.append(cpg)
                except Exception as e:
                    logging.warning(f"Error listing {resource_type}: {str(e)}")
                
            # Delete all found resources
            for resource in resources:
                try:
                    if delete_resource(rds_client, resource_type, resource):
                        deleted_count += 1
                except Exception as e:
                    logging.warning(f"Error deleting {resource_type}: {str(e)}")
        
        except Exception as e:
            logging.warning(f"Error in cleanup for {resource_type}: {str(e)}")
    
    logging.info(f"Force cleanup complete. Submitted deletion for {deleted_count} resources.")
    return deleted_count 