package util

import (
	"context"
	"fmt"
	"sync"

	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
)

var (
	errUnknownParameter         = fmt.Errorf("unknown parameter")
	errUnmodifiableParameter    = fmt.Errorf("parameter is not modifiable")
)

func NewErrUnknownParameter(name string) error {
	// This is a terminal error because unless the user removes this parameter
	// from their list of parameter overrides, we will not be able to get the
	// resource into a synced state.
	return ackerr.NewTerminalError(
		fmt.Errorf("%w: %s", errUnknownParameter, name),
	)
}

func NewErrUnmodifiableParameter(name string) error {
	// This is a terminal error because unless the user removes this parameter
	// from their list of parameter overrides, we will not be able to get the
	// resource into a synced state.
	return ackerr.NewTerminalError(
		fmt.Errorf("%w: %s", errUnmodifiableParameter, name),
	)
}

// ComputeParametersDelta compares two Parameter arrays and returns the new
// parameters to add, to update and the parameter identifiers to delete
func ComputeParametersDelta(
	desired map[string]*string,
	latest map[string]*string,
) (map[string]*string, []string) {
	toReset := []string{}
	toModify := map[string]*string{}

	for k, v := range desired {
		if lv, found := latest[k]; !found {
			toModify[k] = v
		} else if !equalStrings(v, lv) {
			toModify[k] = v
		}
	}
	for k := range latest {
		if _, found := desired[k]; !found {
			toReset = append(toReset, k)
		}
	}
	return toModify, toReset
}

// sliceStringChunks splits a supplied slice of string pointers into multiple
// slices of string pointers of a given size.
func SliceStringChunks(
	input []string,
	chunkSize int,
) [][]string {
	var chunks [][]string
	for {
		if len(input) == 0 {
			break
		}

		if len(input) < chunkSize {
			chunkSize = len(input)
		}

		chunks = append(chunks, input[0:chunkSize])
		input = input[chunkSize:]
	}

	return chunks
}

// mapStringChunks splits a supplied map of string pointers into multiple
// slices of maps of string pointers of a given size.
func MapStringChunks(
	input map[string]*string,
	chunkSize int,
) []map[string]*string {
	var chunks []map[string]*string
	chunk := map[string]*string{}
	idx := 0
	for k, v := range input {
		if idx < chunkSize {
			chunk[k] = v
			idx++
		} else {
			// reset the chunker
			chunks = append(chunks, chunk)
			chunk = map[string]*string{}
			idx = 0
		}
	}
	chunks = append(chunks, chunk)

	return chunks
}

// *** Caching

type ParamMeta struct {
	IsModifiable bool
	IsDynamic    bool
}

// metaFetcher is the functor we pass to the paramMetaCache that allows it to
// fetch engine default parameter information
type metaFetcher func(ctx context.Context, family string) (map[string]ParamMeta, error)

// paramMetaCache stores information about a parameter for a DB parameter group
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

// get retrieves the metadata for a named parameter group family and parameter
// name.
func (c *ParamMetaCache) Get(
	ctx context.Context,
	family string,
	name string,
	fetcher metaFetcher,
) (*ParamMeta, error) {
	var err error
	var found bool
	var metas map[string]ParamMeta
	var meta ParamMeta

	// We need to release the lock right after the read operation, because
	// loadFamilly will might call a writeLock at L619
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
	fetcher metaFetcher,
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