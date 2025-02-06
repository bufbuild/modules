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
	"fmt"
	"os"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/diff"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
)

type manifestDiff struct {
	pathsAdded          map[string]bufcas.FileNode
	pathsRenamed        map[string]fileDiff
	pathsRemoved        map[string]bufcas.FileNode
	pathsChangedContent map[string]fileDiff
}

type fileDiff struct {
	from bufcas.FileNode
	to   bufcas.FileNode
	diff string
}

func newManifestDiff() *manifestDiff {
	return &manifestDiff{
		pathsAdded:          make(map[string]bufcas.FileNode),
		pathsRenamed:        make(map[string]fileDiff),
		pathsRemoved:        make(map[string]bufcas.FileNode),
		pathsChangedContent: make(map[string]fileDiff),
	}
}

func buildManifestDiff(
	ctx context.Context,
	from bufcas.Manifest,
	to bufcas.Manifest,
	bucket storage.ReadBucket,
) (*manifestDiff, error) {
	var (
		diff                 = newManifestDiff()
		digestToAddedPaths   = make(map[string][]string)
		digestToRemovedPaths = make(map[string][]string)
	)
	// removed and changed
	for _, fromNode := range from.FileNodes() {
		path := fromNode.Path()
		toNode := to.GetFileNode(path)
		if toNode == nil {
			diff.pathsRemoved[path] = fromNode
			digestToRemovedPaths[fromNode.Digest().String()] = append(digestToRemovedPaths[path], path)
			continue
		}
		if bufcas.DigestEqual(fromNode.Digest(), toNode.Digest()) {
			continue // no changes
		}
		diffString, err := calculateFileNodeDiff(ctx, fromNode, toNode, bucket)
		if err != nil {
			return nil, fmt.Errorf("calculate file node diff: %w", err)
		}
		diff.pathsChangedContent[path] = fileDiff{
			from: fromNode,
			to:   toNode,
			diff: diffString,
		}
	}
	// added
	for _, toNode := range to.FileNodes() {
		path := toNode.Path()
		if from.GetFileNode(path) == nil {
			diff.pathsAdded[path] = toNode
			digestToAddedPaths[toNode.Digest().String()] = append(digestToAddedPaths[path], path)
		}
	}
	// renamed: defined as digests present both in pathsRemoved and pathsAdded but under different
	// paths
	//
	// We'll keep track of the paths that we matched as renames to later remove them from the
	// added/removed maps
	var (
		matchedRemovedPaths []string
		matchedAddedPaths   []string
	)
	for digest, removedPaths := range digestToRemovedPaths {
		// removedPaths and addedPaths should be lists with no items in common since they're recorded
		// only for added and removed nodes exclusively
		addedPaths, digestHasAddedPaths := digestToAddedPaths[digest]
		if !digestHasAddedPaths {
			continue
		}
		for _, removedPath := range removedPaths {
			if len(addedPaths) == 0 {
				continue
			}
			// both lists are sorted by path, we can always take out the first one to match
			var addedPath string
			addedPaths, addedPath = removeSliceItem(addedPaths, 0)
			matchedRemovedPaths = append(matchedRemovedPaths, removedPath)
			matchedAddedPaths = append(matchedAddedPaths, addedPath)
			diff.pathsRenamed[removedPath] = fileDiff{
				from: from.GetFileNode(removedPath),
				to:   to.GetFileNode(addedPath),
			}
		}
	}
	// delete the matches
	for _, matchedRemovedPath := range matchedRemovedPaths {
		delete(diff.pathsRemoved, matchedRemovedPath)
	}
	for _, matchedAddedPath := range matchedAddedPaths {
		delete(diff.pathsAdded, matchedAddedPath)
	}
	return diff, nil
}

func (d *manifestDiff) printText() {
	os.Stdout.WriteString(fmt.Sprintf(
		"%d files changed: %d removed, %d renamed, %d added, %d changed content\n",
		len(d.pathsRemoved)+len(d.pathsRenamed)+len(d.pathsAdded)+len(d.pathsChangedContent),
		len(d.pathsRemoved),
		len(d.pathsRenamed),
		len(d.pathsAdded),
		len(d.pathsChangedContent),
	))
	if len(d.pathsRemoved) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("Files removed:\n\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.pathsRemoved)
		for _, path := range sortedPaths {
			os.Stdout.WriteString("- " + d.pathsRemoved[path].String() + "\n")
		}
	}
	if len(d.pathsRenamed) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("Files renamed:\n\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.pathsRenamed)
		for _, path := range sortedPaths {
			os.Stdout.WriteString("- " + d.pathsRenamed[path].from.String() + "\n")
			os.Stdout.WriteString("+ " + d.pathsRenamed[path].to.String() + "\n")
		}
	}
	if len(d.pathsAdded) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("Files added:\n\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.pathsAdded)
		for _, path := range sortedPaths {
			os.Stdout.WriteString("+ " + d.pathsAdded[path].String() + "\n")
		}
	}
	if len(d.pathsChangedContent) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("Files changed content:\n\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.pathsChangedContent)
		for _, path := range sortedPaths {
			fnDiff := d.pathsChangedContent[path]
			os.Stdout.WriteString(fnDiff.diff + "\n")
		}
	}
}

func (d *manifestDiff) printMarkdown() {
	os.Stdout.WriteString(fmt.Sprintf(
		"> _%d files changed: %d removed, %d renamed, %d added, %d changed content_\n",
		len(d.pathsRemoved)+len(d.pathsRenamed)+len(d.pathsAdded)+len(d.pathsChangedContent),
		len(d.pathsRemoved),
		len(d.pathsRenamed),
		len(d.pathsAdded),
		len(d.pathsChangedContent),
	))
	if len(d.pathsRemoved) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("# Files removed:\n\n")
		os.Stdout.WriteString("```diff\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.pathsRemoved)
		for _, path := range sortedPaths {
			os.Stdout.WriteString("- " + d.pathsRemoved[path].String() + "\n")
		}
		os.Stdout.WriteString("```\n")
	}
	if len(d.pathsRenamed) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("# Files renamed:\n\n")
		os.Stdout.WriteString("```diff\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.pathsRenamed)
		for _, path := range sortedPaths {
			os.Stdout.WriteString("- " + d.pathsRenamed[path].from.String() + "\n")
			os.Stdout.WriteString("+ " + d.pathsRenamed[path].to.String() + "\n")
		}
		os.Stdout.WriteString("```\n")
	}
	if len(d.pathsAdded) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("# Files added:\n\n")
		os.Stdout.WriteString("```diff\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.pathsAdded)
		for _, path := range sortedPaths {
			os.Stdout.WriteString("+ " + d.pathsAdded[path].String() + "\n")
		}
		os.Stdout.WriteString("```\n")
	}
	if len(d.pathsChangedContent) > 0 {
		os.Stdout.WriteString("\n")
		os.Stdout.WriteString("# Files changed content:\n\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.pathsChangedContent)
		for _, path := range sortedPaths {
			fdiff := d.pathsChangedContent[path]
			// the path we use here can be from/to, is the same, what changed was the content.
			os.Stdout.WriteString("## `" + fdiff.from.Path() + "`:\n")
			os.Stdout.WriteString(
				"```diff\n" +
					fdiff.diff + "\n" +
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

// removeSliceItem returns the slice with the item in index i removed, and the removed item.
func removeSliceItem[T any](s []T, i int) ([]T, T) {
	item := s[i]
	return append(s[:i], s[i+1:]...), item
}
