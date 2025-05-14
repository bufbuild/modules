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
	"bytes"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/modules/private/bufpkg/bufstate"
	"go.uber.org/multierr"
)

const (
	rootSyncDirFlagName = "root-sync-dir"
	srcDirFlagName      = "src-dir"
	ownerFlagName       = "owner"
	repoFlagName        = "repo"
	refFlagName         = "ref"
)

type command struct {
	rootSyncDir string
	srcDir      string
	owner       string
	repo        string
	ref         string
}

func newCmd(
	rootSyncDir string,
	srcDir string,
	owner string,
	repo string,
	modRef string,
) (*command, error) {
	var err error
	if len(rootSyncDir) == 0 {
		err = multierr.Append(err, fmt.Errorf("%s is required", rootSyncDirFlagName))
	}
	if len(srcDir) == 0 {
		err = multierr.Append(err, fmt.Errorf("%s is required", srcDirFlagName))
	}
	if len(owner) == 0 {
		err = multierr.Append(err, fmt.Errorf("%s is required", ownerFlagName))
	}
	if len(repo) == 0 {
		err = multierr.Append(err, fmt.Errorf("%s is required", repoFlagName))
	}
	if len(modRef) == 0 {
		err = multierr.Append(err, fmt.Errorf("%s is required", refFlagName))
	}
	if err != nil {
		return nil, err
	}
	return &command{
		rootSyncDir: rootSyncDir,
		srcDir:      srcDir,
		owner:       owner,
		repo:        repo,
		ref:         modRef,
	}, nil
}

func main() {
	var (
		rootSyncDir = flag.String(rootSyncDirFlagName, "", "Root sync directory where all the managed modules live.")
		srcDir      = flag.String(srcDirFlagName, "", "Directory with the managed module source files.")
		owner       = flag.String(ownerFlagName, "", "Managed module owner name.")
		repo        = flag.String(repoFlagName, "", "Managed module repository name.")
		ref         = flag.String(refFlagName, "", "Managed module reference that matches the contents in the source directory.")
	)
	flag.Parse()
	cmd, err := newCmd(
		*rootSyncDir,
		*srcDir,
		*owner,
		*repo,
		*ref,
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cannot run mod processor: %v\n\nusage: modprocessor [flags]\n\n", err)
		flag.PrintDefaults()
		os.Exit(2)
	}
	if err := cmd.run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "modprocessor failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func (c *command) run() error {
	ctx := context.Background()
	manifestDigest, err := c.convertToCAS(ctx)
	if err != nil {
		return fmt.Errorf("convert module to CAS: %w", err)
	}
	stateRW, err := bufstate.NewReadWriter()
	if err != nil {
		return fmt.Errorf("new state read writer: %w", err)
	}
	manifestHexDigest := hex.EncodeToString(manifestDigest.Value())
	if err := stateRW.AppendModuleReference(c.rootSyncDir, c.owner, c.repo, c.ref, manifestHexDigest); err != nil {
		return fmt.Errorf("update mod reference: %w", err)
	}
	return nil
}

// convertToCAS converts all files in the source directory to blobs, and a saves
// them in the module destination directory using its manifest digest hex string
// as filenames.
func (c *command) convertToCAS(ctx context.Context) (bufcas.Digest, error) {
	storageosProvider := storageos.NewProvider()
	bucket, err := storageosProvider.NewReadWriteBucket(c.srcDir)
	if err != nil {
		return nil, fmt.Errorf("new bucket from buf dir: %w", err)
	}
	fileSet, err := bufcas.NewFileSetForBucket(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("new file set from bucket: %w", err)
	}
	manifestBlob, err := bufcas.ManifestToBlob(fileSet.Manifest())
	if err != nil {
		return nil, fmt.Errorf("manifest to blob: %w", err)
	}
	modSyncDir := filepath.Join(c.rootSyncDir, c.owner, c.repo, "cas")
	// mkdir directory in case this is the first time a reference is being synced for this module.
	if err := os.MkdirAll(modSyncDir, 0755); err != nil {
		return nil, fmt.Errorf("make module sync cas dir: %w", err)
	}
	// TODO: parallelize
	for _, blob := range append([]bufcas.Blob{manifestBlob}, fileSet.BlobSet().Blobs()...) {
		if err := writeBlobInDir(blob, modSyncDir); err != nil {
			hexDigest := hex.EncodeToString(blob.Digest().Value())
			return nil, fmt.Errorf("write blob %q to file: %w", hexDigest, err)
		}
	}
	return manifestBlob.Digest(), nil
}

// writeBlobInDir takes a blob and writes its content to a file named as its
// digest hex in the given directory (only if it doesn't exist already).
func writeBlobInDir(blob bufcas.Blob, dir string) (retErr error) {
	fileName := hex.EncodeToString(blob.Digest().Value())
	filePath := filepath.Join(dir, fileName)
	_, err := os.Stat(filePath)
	if err == nil {
		// no error, file exists
		return nil
	}
	// some error
	if !os.IsNotExist(err) {
		// some error different from "not exists"
		return fmt.Errorf("stat file: %w", err)
	}
	// error is "not exists", we'll create it
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("close file: %w", err))
		}
	}()
	if _, err := io.Copy(file, bytes.NewReader(blob.Content())); err != nil {
		return fmt.Errorf("copy to file: %w", err)
	}
	return nil
}
