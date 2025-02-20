// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package util

import (
	"context"
	"sync"
)

// ParamMeta stores metadata about a parameter in a parameter group
type ParamMeta struct {
	IsModifiable bool
	IsDynamic    bool
}

// MetaFetcher is the functor we pass to the paramMetaCache that allows it to
// fetch engine default parameter information
type MetaFetcher func(ctx context.Context, family string) (map[string]ParamMeta, error)

// ParamMetaCache stores information about a parameter for a DB parameter group
// family. We use this cached information to determine whether a parameter is
// statically or dynamically defined (whether changes can be applied
// immediately or pending a reboot) and whether a parameter is modifiable.
//
// Keeping things super simple for now and not adding any TTL or expiration
// behaviour to the cache. Engine defaults are pretty static information...
type ParamMetaCache struct {
	sync.RWMutex
	Hits   uint64
	Misses uint64
	Cache  map[string]map[string]ParamMeta
}

// ClearFamily removes the cached parameter information for a specific family
func (c *ParamMetaCache) ClearFamily(family string) {
	c.Lock()
	defer c.Unlock()
	delete(c.Cache, family)
}

// Get retrieves the metadata for a named parameter group family and parameter
// name.
func (c *ParamMetaCache) Get(
	ctx context.Context,
	family string,
	name string,
	fetcher MetaFetcher,
) (*ParamMeta, error) {
	var err error
	var found bool
	var metas map[string]ParamMeta
	var meta ParamMeta

	// We need to release the lock right after the read operation, because
	// loadFamily might call a writeLock below
	c.RLock()
	metas, found = c.Cache[family]
	c.RUnlock()

	if !found {
		c.Misses++
		metas, err = c.loadFamily(ctx, family, fetcher)
		if err != nil {
			return nil, err
		}
	}
	meta, found = metas[name]
	if !found {
		// Clear the cache for this family when a parameter is not found
		// This ensures the next reconciliation will fetch fresh metadata
		c.ClearFamily(family)
		return nil, NewErrUnknownParameter(name)
	}
	c.Hits++
	return &meta, nil
}

// loadFamily fetches parameter information from the AWS RDS
// DescribeEngineDefaultParameters API and caches that information.
func (c *ParamMetaCache) loadFamily(
	ctx context.Context,
	family string,
	fetcher MetaFetcher,
) (map[string]ParamMeta, error) {
	familyMeta, err := fetcher(ctx, family)
	if err != nil {
		return nil, err
	}
	c.Lock()
	defer c.Unlock()
	c.Cache[family] = familyMeta
	return familyMeta, nil
}
