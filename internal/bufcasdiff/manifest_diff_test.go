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
	"encoding/hex"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/cas"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden test files")

func TestManifestDiff(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	casBucket, mFrom, mTo := prepareDiffCASBucket(ctx, t)
	mdiff, err := buildManifestDiff(ctx, mFrom, mTo, casBucket)
	require.NoError(t, err)
	require.NotNil(t, mdiff)
	t.Run("removed", func(t *testing.T) {
		t.Parallel()
		expectedRemovedPaths := map[string]struct{}{
			"to_remove.txt": {},
		}
		assert.Len(t, mdiff.pathsRemoved, len(expectedRemovedPaths))
		for expectedRemovedPath := range expectedRemovedPaths {
			expected := mFrom.GetFileNode(expectedRemovedPath)
			require.NotNil(t, expected)
			actual, present := mdiff.pathsRemoved[expectedRemovedPath]
			require.True(t, present)
			assert.Equal(t, expectedRemovedPath, actual.Path())
			assert.True(t, cas.DigestEqual(expected.Digest(), actual.Digest()))
		}
	})
	t.Run("renamed", func(t *testing.T) {
		t.Parallel()
		// two directories in testdata are renamed, and the content in all files of each group is the
		// same, so this test actually makes sure that all of them are matched (even with the same
		// digest), and that the order is trying to be kept.
		expectedRenamedPaths := map[string]string{
			"to_rename_bar/1.txt": "renamed_bar/1.txt",
			"to_rename_bar/2.txt": "renamed_bar/2.txt",
			"to_rename_bar/3.txt": "renamed_bar/3.txt",
			"to_rename_foo/1.txt": "renamed_foo/1.txt",
			"to_rename_foo/2.txt": "renamed_foo/2.txt",
			"to_rename_foo/3.txt": "renamed_foo/3.txt",
		}
		assert.Len(t, mdiff.pathsRenamed, len(expectedRenamedPaths))
		for fromPath, toPath := range expectedRenamedPaths {
			expected := mFrom.GetFileNode(fromPath)
			require.NotNil(t, expected)
			actual, present := mdiff.pathsRenamed[fromPath]
			require.True(t, present)
			assert.Equal(t, fromPath, actual.from.Path())
			assert.Equal(t, toPath, actual.to.Path())
			assert.True(t, cas.DigestEqual(expected.Digest(), actual.from.Digest()))
			assert.True(t, cas.DigestEqual(actual.from.Digest(), actual.to.Digest()))
			assert.Empty(t, actual.diff)
		}
	})
	t.Run("added", func(t *testing.T) {
		t.Parallel()
		expectedAddedPaths := map[string]struct{}{
			"added.txt": {},
		}
		assert.Len(t, mdiff.pathsAdded, len(expectedAddedPaths))
		for expectedAddedPath := range expectedAddedPaths {
			expected := mTo.GetFileNode(expectedAddedPath)
			require.NotNil(t, expected)
			actual, present := mdiff.pathsAdded[expectedAddedPath]
			require.True(t, present)
			assert.Equal(t, expectedAddedPath, actual.Path())
			assert.True(t, cas.DigestEqual(expected.Digest(), actual.Digest()))
		}
	})
	t.Run("changed_content", func(t *testing.T) {
		t.Parallel()
		expectedChangedContentPaths := map[string]struct{}{
			"changes.txt": {},
		}
		assert.Len(t, mdiff.pathsChangedContent, len(expectedChangedContentPaths))
		for expectedChangedContentPath := range expectedChangedContentPaths {
			expected := mTo.GetFileNode(expectedChangedContentPath)
			require.NotNil(t, expected)
			actual, present := mdiff.pathsChangedContent[expectedChangedContentPath]
			require.True(t, present)
			assert.Equal(t, actual.from.Path(), actual.to.Path())
			assert.False(t, cas.DigestEqual(actual.from.Digest(), actual.to.Digest()))
			assert.NotEmpty(t, actual.diff)
		}
	})
}

func TestManifestDiffString(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	casBucket, mFrom, mTo := prepareDiffCASBucket(ctx, t)
	mdiff, err := buildManifestDiff(ctx, mFrom, mTo, casBucket)
	require.NoError(t, err)
	require.NotNil(t, mdiff)

	type testCase struct {
		name      string
		format    ManifestDiffOutputFormat
		extension string
	}
	for _, tc := range []testCase{
		{
			name:      "text",
			format:    ManifestDiffOutputFormatText,
			extension: ".txt",
		},
		{
			name:      "markdown",
			format:    ManifestDiffOutputFormatMarkdown,
			extension: ".md",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mdiff.String(tc.format)
			golden := filepath.Join("testdata", "manifest_diff", tc.name+".golden"+tc.extension)
			if *update {
				require.NoError(t, os.WriteFile(golden, []byte(got), 0600))
			}
			want, err := os.ReadFile(golden)
			require.NoError(t, err)
			assert.Equal(t, string(want), got)
		})
	}
}

func prepareDiffCASBucket(ctx context.Context, t *testing.T) (
	storage.ReadBucket,
	cas.Manifest,
	cas.Manifest,
) {
	t.Helper()
	casBucket := storagemem.NewReadWriteBucket()
	casWrite := func(dirpath string) cas.Manifest {
		testFiles, err := storageos.NewProvider().NewReadWriteBucket("testdata/manifest_diff/" + dirpath)
		require.NoError(t, err)
		fileSet, err := cas.NewFileSetForBucket(ctx, testFiles)
		require.NoError(t, err)
		mBlob, err := cas.ManifestToBlob(fileSet.Manifest())
		require.NoError(t, err)
		blobsToPut := append(fileSet.BlobSet().Blobs(), mBlob)
		for _, blob := range blobsToPut {
			require.NoError(t, storage.PutPath(ctx, casBucket, hex.EncodeToString(blob.Digest().Value()), blob.Content()))
		}
		return fileSet.Manifest()
	}
	var (
		mFrom = casWrite("from")
		mTo   = casWrite("to")
	)
	return casBucket, mFrom, mTo
}
