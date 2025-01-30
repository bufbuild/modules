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
	"flag"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/modules/private/bufpkg/bufstate"
	"github.com/pmezard/go-difflib/difflib"
)

type command struct {
	from string
	to   string
}

func main() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "usage: diff <from> <to>")
		flag.PrintDefaults()
		os.Exit(2)
	}
	if flag.Args()[0] == flag.Args()[1] {
		_, _ = fmt.Fprintf(os.Stderr, "<from> and <to> cannot be the same")
		flag.PrintDefaults()
		os.Exit(2)
	}
	cmd := &command{
		from: flag.Args()[0],
		to:   flag.Args()[1],
	}
	if err := cmd.run(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "diff: %v\n", err)
		os.Exit(1)
	}
}

func (c *command) run(ctx context.Context) error {
	// first, attempt to match from/to as module references in a state file in the same directory
	// where the command is run
	bucket, err := storageos.NewProvider().NewReadWriteBucket(".")
	if err != nil {
		return fmt.Errorf("new rw bucket: %w", err)
	}
	modStateReader, err := bucket.Get(ctx, bufstate.ModStateFileName)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("read module state file: %w", err)
		}
		// if the state file does not exist, we assume we are in the cas directory, and that from/to are
		// the manifest paths
		return diffFromCASDirectory(ctx, bucket, c.from, c.to)
	}
	// state file was found, attempt to parse it and use its digests
	rw, err := bufstate.NewReadWriter()
	if err != nil {
		return fmt.Errorf("new state rw: %w", err)
	}
	modState, err := rw.ReadModStateFile(modStateReader)
	if err != nil {
		return fmt.Errorf("read module state: %w", err)
	}
	var (
		fromManifestPath string
		toManifestPath   string
	)
	for _, ref := range modState.GetReferences() {
		if ref.GetName() == c.from {
			fromManifestPath = ref.GetDigest()
			if toManifestPath != "" {
				break
			}
		} else if ref.GetName() == c.to {
			toManifestPath = ref.GetDigest()
			if fromManifestPath != "" {
				break
			}
		}
	}
	if fromManifestPath == "" {
		return fmt.Errorf("from reference %s not found in the module state file", c.from)
	}
	if toManifestPath == "" {
		return fmt.Errorf("to reference %s not found in the module state file", c.to)
	}
	casBucket, err := storageos.NewProvider().NewReadWriteBucket("cas")
	if err != nil {
		return fmt.Errorf("new rw cas bucket: %w", err)
	}
	return diffFromCASDirectory(ctx, casBucket, fromManifestPath, toManifestPath)
}

func diffFromCASDirectory(
	ctx context.Context,
	bucket storage.ReadBucket,
	fromManifestPath string,
	toManifestPath string,
) error {
	fromManifest, err := readManifest(ctx, bucket, fromManifestPath)
	if err != nil {
		return fmt.Errorf("read manifest from: %w", err)
	}
	toManifest, err := readManifest(ctx, bucket, toManifestPath)
	if err != nil {
		return fmt.Errorf("read manifest to: %w", err)
	}
	diff := newDiff()
	// removed and changed
	for _, fromNode := range fromManifest.FileNodes() {
		path := fromNode.Path()
		toNode := toManifest.GetFileNode(path)
		if toNode == nil {
			diff.removed(path, fromNode)
			continue
		}
		if bufcas.DigestEqual(fromNode.Digest(), toNode.Digest()) {
			continue // no changes
		}
		if err := diff.changedDigest(ctx, bucket, path, fromNode, toNode); err != nil {
			return fmt.Errorf(
				"changed digest from %s to %s: %w",
				fromNode.String(),
				toNode.String(),
				err,
			)
		}
	}
	// added
	for _, toNode := range toManifest.FileNodes() {
		if fromManifest.GetFileNode(toNode.Path()) == nil {
			diff.added(toNode.Path(), toNode)
		}
	}
	fmt.Println(diff.String())
	return nil
}

func readManifest(ctx context.Context, bucket storage.ReadBucket, manifestPath string) (bufcas.Manifest, error) {
	data, err := storage.ReadPath(ctx, bucket, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read path: %w", err)
	}
	m, err := bufcas.ParseManifest(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return m, nil
}

type pathDiff struct {
	from bufcas.FileNode
	to   bufcas.FileNode
	diff string
}

type diff struct {
	addedPaths         map[string]bufcas.FileNode
	removedPaths       map[string]bufcas.FileNode
	changedDigestPaths map[string]pathDiff
}

func newDiff() *diff {
	return &diff{
		addedPaths:         make(map[string]bufcas.FileNode),
		removedPaths:       make(map[string]bufcas.FileNode),
		changedDigestPaths: make(map[string]pathDiff),
	}
}

func (d *diff) added(path string, node bufcas.FileNode) {
	d.addedPaths[path] = node
}

func (d *diff) removed(path string, node bufcas.FileNode) {
	d.removedPaths[path] = node
}

func (d *diff) changedDigest(
	ctx context.Context,
	bucket storage.ReadBucket,
	path string,
	from bufcas.FileNode,
	to bufcas.FileNode,
) error {
	fromData, err := storage.ReadPath(ctx, bucket, fileNodePath(from))
	if err != nil {
		return fmt.Errorf("read from changed path: %w", err)
	}
	toData, err := storage.ReadPath(ctx, bucket, fileNodePath(to))
	if err != nil {
		return fmt.Errorf("read to changed path: %w", err)
	}
	udiff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(fromData)),
		B:        difflib.SplitLines(string(toData)),
		FromFile: from.Digest().String(),
		ToFile:   to.Digest().String(),
		Context:  2,
	}
	diff, err := difflib.GetUnifiedDiffString(udiff)
	if err != nil {
		return fmt.Errorf("get unified diff: %w", err)
	}
	d.changedDigestPaths[path] = pathDiff{
		from: from,
		to:   to,
		diff: diff,
	}
	return nil
}

func (d *diff) String() string {
	var sb strings.Builder
	if len(d.removedPaths) > 0 {
		sb.WriteString("# Removed:\n\n")
		sb.WriteString("```diff\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.removedPaths)
		for _, path := range sortedPaths {
			sb.WriteString("- " + d.removedPaths[path].String() + "\n")
		}
		sb.WriteString("```\n")
		sb.WriteString("\n")
	}
	if len(d.addedPaths) > 0 {
		sb.WriteString("# Added:\n\n")
		sb.WriteString("```diff\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.addedPaths)
		for _, path := range sortedPaths {
			sb.WriteString("+ " + d.addedPaths[path].String() + "\n")
		}
		sb.WriteString("```\n")
		sb.WriteString("\n")
	}
	if len(d.changedDigestPaths) > 0 {
		sb.WriteString("# Changed content:\n\n")
		sortedPaths := slicesext.MapKeysToSortedSlice(d.changedDigestPaths)
		for _, path := range sortedPaths {
			pd := d.changedDigestPaths[path]
			sb.WriteString("## `" + pd.from.Path() + "`:\n")
			sb.WriteString(
				"```diff\n" +
					pd.diff + "\n" +
					"```\n",
			)
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf(
		"Total changes: %d (%d deletions, %d additions, %d changed)\n",
		len(d.removedPaths)+len(d.addedPaths)+len(d.changedDigestPaths),
		len(d.removedPaths),
		len(d.addedPaths),
		len(d.changedDigestPaths),
	))
	return sb.String()
}

func fileNodePath(n bufcas.FileNode) string {
	return hex.EncodeToString(n.Digest().Value())
}
