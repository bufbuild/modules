// Copyright 2021-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bufcasdiff

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/cas"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/modules/private/bufpkg/bufstate"
)

// DiffModuleDirectory computes the diff between two refs or two digests in the module directory at
// dirPath.
//
// If a state.json file is present, from/to are resolved as ref names against it, otherwise, dirPath
// is treated as a CAS directory and from/to are manifest filenames directly.
func DiffModuleDirectory(
	ctx context.Context,
	dirPath string,
	from string,
	to string,
) (*ManifestDiff, error) {
	if from == to {
		return newManifestDiff(), nil
	}
	bucket, err := storageos.NewProvider().NewReadWriteBucket(dirPath)
	if err != nil {
		return nil, fmt.Errorf("new rw bucket: %w", err)
	}
	moduleStateReader, err := bucket.Get(ctx, bufstate.ModStateFileName)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("read module state file: %w", err)
		}
		// No state.json — dirPath is a CAS directory and from/to are manifest filenames.
		return calculateDiffFromCASDirectory(ctx, bucket, from, to)
	}
	// state file was found, attempt to parse it and match from/to with its references
	stateRW, err := bufstate.NewReadWriter()
	if err != nil {
		return nil, fmt.Errorf("new state rw: %w", err)
	}
	moduleState, err := stateRW.ReadModStateFile(moduleStateReader)
	if err != nil {
		return nil, fmt.Errorf("read module state: %w", err)
	}
	var (
		fromManifestPath string
		toManifestPath   string
	)
	for _, ref := range moduleState.GetReferences() {
		if ref.GetName() == from {
			fromManifestPath = ref.GetDigest()
			if toManifestPath != "" {
				break
			}
		} else if ref.GetName() == to {
			toManifestPath = ref.GetDigest()
			if fromManifestPath != "" {
				break
			}
		}
	}
	if fromManifestPath == "" {
		return nil, fmt.Errorf("from reference %s not found in the module state file", from)
	}
	if toManifestPath == "" {
		return nil, fmt.Errorf("to reference %s not found in the module state file", to)
	}
	if fromManifestPath == toManifestPath {
		return newManifestDiff(), nil
	}
	casBucket, err := storageos.NewProvider().NewReadWriteBucket(filepath.Join(dirPath, "cas"))
	if err != nil {
		return nil, fmt.Errorf("new rw cas bucket: %w", err)
	}
	return calculateDiffFromCASDirectory(ctx, casBucket, fromManifestPath, toManifestPath)
}

// calculateDiffFromCASDirectory takes the cas bucket, and the from/to manifest paths to calculate a
// diff.
func calculateDiffFromCASDirectory(
	ctx context.Context,
	casBucket storage.ReadBucket,
	fromManifestPath string,
	toManifestPath string,
) (*ManifestDiff, error) {
	if fromManifestPath == toManifestPath {
		return newManifestDiff(), nil
	}
	fromManifest, err := readManifest(ctx, casBucket, fromManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest from: %w", err)
	}
	toManifest, err := readManifest(ctx, casBucket, toManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest to: %w", err)
	}
	return buildManifestDiff(ctx, fromManifest, toManifest, casBucket)
}

func readManifest(ctx context.Context, bucket storage.ReadBucket, manifestPath string) (cas.Manifest, error) {
	data, err := storage.ReadPath(ctx, bucket, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read path: %w", err)
	}
	m, err := cas.ParseManifest(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return m, nil
}
