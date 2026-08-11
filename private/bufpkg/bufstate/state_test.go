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

package bufstate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLatestModuleDigest(t *testing.T) {
	t.Parallel()
	readWriter, err := NewReadWriter()
	require.NoError(t, err)
	t.Run("noStateFile", func(t *testing.T) {
		t.Parallel()
		latestDigest, err := readWriter.LatestModuleDigest(t.TempDir(), "acme", "widgets")
		require.NoError(t, err)
		require.Empty(t, latestDigest)
	})
	t.Run("appendedReferences", func(t *testing.T) {
		t.Parallel()
		rootSyncDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(rootSyncDir, "acme", "widgets"), 0755))
		require.NoError(t, readWriter.AppendModuleReference(rootSyncDir, "acme", "widgets", "commit1", "digest1"))
		require.NoError(t, readWriter.AppendModuleReference(rootSyncDir, "acme", "widgets", "commit2", "digest2"))
		latestDigest, err := readWriter.LatestModuleDigest(rootSyncDir, "acme", "widgets")
		require.NoError(t, err)
		require.Equal(t, "digest2", latestDigest)
	})
}
