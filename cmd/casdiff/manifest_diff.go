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

package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/diff"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
)

type manifestDiff struct {
	added   map[string]bufcas.FileNode
	removed map[string]bufcas.FileNode
	changed map[string]fileChangedDiff
}

type fileChangedDiff struct {
	path string
	from bufcas.Digest
	to   bufcas.Digest
	diff string
}

func newManifestDiff() *manifestDiff {
	return &manifestDiff{
		added:   make(map[string]bufcas.FileNode),
		removed: make(map[string]bufcas.FileNode),
		changed: make(map[string]fileChangedDiff),
	}
}

func buildManifestDiff(
	ctx context.Context,
	from bufcas.Manifest,
	to bufcas.Manifest,
	bucket storage.ReadBucket,
) (*manifestDiff, error) {
	diff := newManifestDiff()
	// removed and changed
	for _, fromNode := range from.FileNodes() {
		path := fromNode.Path()
		toNode := to.GetFileNode(path)
		if toNode == nil {
			diff.removed[path] = fromNode
			continue
		}
		if bufcas.DigestEqual(fromNode.Digest(), toNode.Digest()) {
			continue // no changes
		}
		if err := diff.newChangedPath(ctx, bucket, path, fromNode, toNode); err != nil {
			return nil, fmt.Errorf(
				"changed digest from %s to %s: %w",
				fromNode.String(),
				toNode.String(),
				err,
			)
		}
	}
	// added
	for _, toNode := range to.FileNodes() {
		if from.GetFileNode(toNode.Path()) == nil {
			diff.added[toNode.Path()] = toNode
		}
	}
	return diff, nil
}

func (d *manifestDiff) newChangedPath(
	ctx context.Context,
	bucket storage.ReadBucket,
	path string,
	from bufcas.FileNode,
	to bufcas.FileNode,
) error {
	if from.Path() != to.Path() {
		return errors.New("from and to paths are different")
	}
	diffString, err := calculateFileNodeDiff(ctx, from, to, bucket)
	if err != nil {
		return fmt.Errorf("calculate digest diff: %w", err)
	}
	d.changed[path] = fileChangedDiff{
		path: from.Path(),
		from: from.Digest(),
		to:   to.Digest(),
		diff: diffString,
	}
	return nil
}

func (d *manifestDiff) printText() {
	os.Stdout.WriteString(fmt.Sprintf(
		"%d files changed: %d removed, %d added, %d changed content\n",
		len(d.removed)+len(d.added)+len(d.changed),
		len(d.removed),
		len(d.added),
		len(d.changed),
	))
	if len(d.removed) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("Files removed:\n\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.removed)
		for _, path := range sortedPaths {
			os.Stdout.WriteString("- " + d.removed[path].String() + "\n")
		}
	}
	if len(d.added) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("Files added:\n\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.added)
		for _, path := range sortedPaths {
			os.Stdout.WriteString("+ " + d.added[path].String() + "\n")
		}
	}
	if len(d.changed) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("Files changed content:\n\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.changed)
		for _, path := range sortedPaths {
			fnDiff := d.changed[path]
			os.Stdout.WriteString(fnDiff.diff + "\n")
		}
	}
}

func (d *manifestDiff) printMarkdown() {
	os.Stdout.WriteString(fmt.Sprintf(
		"> _%d files changed: %d removed, %d added, %d changed content_\n",
		len(d.removed)+len(d.added)+len(d.changed),
		len(d.removed),
		len(d.added),
		len(d.changed),
	))
	if len(d.removed) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("# Files removed:\n\n")
		os.Stdout.WriteString("```diff\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.removed)
		for _, path := range sortedPaths {
			os.Stdout.WriteString("- " + d.removed[path].String() + "\n")
		}
		os.Stdout.WriteString("```\n")
	}
	if len(d.added) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("# Files added:\n\n")
		os.Stdout.WriteString("```diff\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.added)
		for _, path := range sortedPaths {
			os.Stdout.WriteString("+ " + d.added[path].String() + "\n")
		}
		os.Stdout.WriteString("```\n")
	}
	if len(d.changed) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("# Files changed content:\n\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.changed)
		for _, path := range sortedPaths {
			fnDiff := d.changed[path]
			os.Stdout.WriteString("## `" + fnDiff.path + "`:\n")
			os.Stdout.WriteString(
				"```diff\n" +
					fnDiff.diff + "\n" +
					"```\n",
			)
		}
	}
}

func calculateFileNodeDiff(
	ctx context.Context,
	from bufcas.FileNode,
	to bufcas.FileNode,
	bucket storage.ReadBucket,
) (string, error) {
	if from.Path() == to.Path() && bufcas.DigestEqual(from.Digest(), to.Digest()) {
		return "", nil
	}
	var (
		fromFilePath = hex.EncodeToString(from.Digest().Value())
		toFilePath   = hex.EncodeToString(to.Digest().Value())
	)
	fromData, err := storage.ReadPath(ctx, bucket, fromFilePath)
	if err != nil {
		return "", fmt.Errorf("read from path: %w", err)
	}
	toData, err := storage.ReadPath(ctx, bucket, toFilePath)
	if err != nil {
		return "", fmt.Errorf("read to path: %w", err)
	}
	diffData, err := diff.Diff(
		ctx,
		fromData,
		toData,
		from.String(),
		to.String(),
		diff.DiffWithSuppressCommands(),
		diff.DiffWithSuppressTimestamps(),
	)
	if err != nil {
		return "", fmt.Errorf("diff: %w", err)
	}
	return string(diffData), nil
}
