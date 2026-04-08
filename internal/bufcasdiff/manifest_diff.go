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
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/pkg/cas"
	"github.com/bufbuild/buf/private/pkg/diff"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// ManifestDiffOutputFormat is the format in which a manifest diff can output its results.
type ManifestDiffOutputFormat int

const (
	ManifestDiffOutputFormatText = iota + 1
	ManifestDiffOutputFormatMarkdown
)

// ManifestDiff represents a change in between two CAS manifests.
type ManifestDiff struct {
	pathsAdded          map[string]cas.FileNode
	pathsRenamed        map[string]fileDiff
	pathsRemoved        map[string]cas.FileNode
	pathsChangedContent map[string]fileDiff
}

type fileDiff struct {
	from cas.FileNode
	to   cas.FileNode
	diff string
}

func newManifestDiff() *ManifestDiff {
	return &ManifestDiff{
		pathsAdded:          make(map[string]cas.FileNode),
		pathsRenamed:        make(map[string]fileDiff),
		pathsRemoved:        make(map[string]cas.FileNode),
		pathsChangedContent: make(map[string]fileDiff),
	}
}

func buildManifestDiff(
	ctx context.Context,
	from cas.Manifest,
	to cas.Manifest,
	bucket storage.ReadBucket,
) (*ManifestDiff, error) {
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
			digestToRemovedPaths[fromNode.Digest().String()] = append(digestToRemovedPaths[fromNode.Digest().String()], path)
			continue
		}
		if cas.DigestEqual(fromNode.Digest(), toNode.Digest()) {
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
			digestToAddedPaths[toNode.Digest().String()] = append(digestToAddedPaths[toNode.Digest().String()], path)
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

// Summary returns a manifest diff summary in the shape of:
//
// %d files changed: %d removed, %d renamed, %d added, %d changed content.
func (d *ManifestDiff) Summary() string {
	return fmt.Sprintf(
		"%d files changed: %d removed, %d renamed, %d added, %d changed content.",
		len(d.pathsRemoved)+len(d.pathsRenamed)+len(d.pathsAdded)+len(d.pathsChangedContent),
		len(d.pathsRemoved),
		len(d.pathsRenamed),
		len(d.pathsAdded),
		len(d.pathsChangedContent),
	)
}

// String returns the diff output in the given format. On invalid or unknown format, this function
// defaults to ManifestDiffOutputFormatText.
func (d *ManifestDiff) String(format ManifestDiffOutputFormat) string {
	var b bytes.Buffer
	isMarkdown := format == ManifestDiffOutputFormatMarkdown
	if isMarkdown {
		b.WriteString("> ")
	}
	b.WriteString(d.Summary() + "\n")
	if len(d.pathsRemoved) > 0 {
		b.WriteString("\n")
		if isMarkdown {
			b.WriteString("# ")
		}
		b.WriteString("Files removed:\n\n")
		if isMarkdown {
			b.WriteString("```diff\n")
		}
		for _, path := range xslices.MapKeysToSortedSlice(d.pathsRemoved) {
			b.WriteString("- " + d.pathsRemoved[path].String() + "\n")
		}
		if isMarkdown {
			b.WriteString("```\n")
		}
	}
	if len(d.pathsRenamed) > 0 {
		b.WriteString("\n")
		if isMarkdown {
			b.WriteString("# ")
		}
		b.WriteString("Files renamed:\n\n")
		if isMarkdown {
			b.WriteString("```diff\n")
		}
		for _, path := range xslices.MapKeysToSortedSlice(d.pathsRenamed) {
			b.WriteString("- " + d.pathsRenamed[path].from.String() + "\n")
			b.WriteString("+ " + d.pathsRenamed[path].to.String() + "\n")
		}
		if isMarkdown {
			b.WriteString("```\n")
		}
	}
	if len(d.pathsAdded) > 0 {
		b.WriteString("\n")
		if isMarkdown {
			b.WriteString("# ")
		}
		b.WriteString("Files added:\n\n")
		if isMarkdown {
			b.WriteString("```diff\n")
		}
		for _, path := range xslices.MapKeysToSortedSlice(d.pathsAdded) {
			b.WriteString("+ " + d.pathsAdded[path].String() + "\n")
		}
		if isMarkdown {
			b.WriteString("```\n")
		}
	}
	if len(d.pathsChangedContent) > 0 { //nolint:nestif // Markdown vs text small differences.
		b.WriteString("\n")
		if isMarkdown {
			b.WriteString("# ")
		}
		b.WriteString("Files changed content:\n\n")
		for _, path := range xslices.MapKeysToSortedSlice(d.pathsChangedContent) {
			fdiff := d.pathsChangedContent[path]
			if isMarkdown {
				b.WriteString("## `" + fdiff.from.Path() + "`:\n")
			} else {
				b.WriteString(fdiff.from.Path() + ":\n")
			}
			if isMarkdown {
				b.WriteString(markdownFencedDiff(fdiff.diff))
			} else {
				b.WriteString(fdiff.diff + "\n")
			}
		}
	}
	return b.String()
}

// markdownFencedDiff wraps content in a ```diff code fence, using a longer fence if the content
// itself contains backtick runs that would break the fence.
func markdownFencedDiff(content string) string {
	fence := "```"
	for strings.Contains(content, fence) {
		fence += "`"
	}
	return fence + "diff\n" + content + "\n" + fence + "\n"
}

func calculateFileNodeDiff(
	ctx context.Context,
	from cas.FileNode,
	to cas.FileNode,
	bucket storage.ReadBucket,
) (string, error) {
	if from.Path() == to.Path() && cas.DigestEqual(from.Digest(), to.Digest()) {
		return "", nil
	}
	var (
		fromFilePath = hex.EncodeToString(from.Digest().Value())
		toFilePath   = hex.EncodeToString(to.Digest().Value())
	)
	fromData, err := storage.ReadPath(ctx, bucket, fromFilePath)
	if err != nil {
		return "", fmt.Errorf("read path from: %w", err)
	}
	toData, err := storage.ReadPath(ctx, bucket, toFilePath)
	if err != nil {
		return "", fmt.Errorf("read path to: %w", err)
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
