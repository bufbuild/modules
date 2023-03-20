// Copyright 2021-2023 Buf Technologies, Inc.
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
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/modules/internal/modules"
	"github.com/bufbuild/modules/private/bufpkg/bufstate"
	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateNewReleaseModules(t *testing.T) {
	t.Parallel()
	t.Run("NewReleases-FromNoPreviousRelease", func(t *testing.T) {
		t.Parallel()
		currentRelease := map[string]string{
			"envoyproxy/envoy": "bb554f53ad8d3a2a2ae4cbd7102a3e20ae00b558",
		}
		got, newContent, err := calculateModulesStates(filepath.Join("testdata/golden/new-release", bufstate.SyncRoot), nil, currentRelease)
		require.NoError(t, err)
		assert.Equal(t, map[string]releaseModuleState{
			"envoyproxy/envoy": {
				status: modules.New,
				references: []*statev1alpha1.ModuleReference{{
					Name:   "bb554f53ad8d3a2a2ae4cbd7102a3e20ae00b558",
					Digest: "dummyManifestDigestEnvoy",
				}},
			}}, got)
		assert.True(t, newContent)
	})
	t.Run("UpdatedRelease-FromPreviousRelease", func(t *testing.T) {
		t.Parallel()
		prevRelease := map[string]string{
			"envoyproxy/envoy": "bb554f53ad8d3a2a2ae4cbd7102a3e20ae00b558",
		}
		currentRelease := map[string]string{
			"envoyproxy/envoy": "7850b6bb6494e3bfc093b1aff20282ab30b67940", // updated

		}
		got, newContent, err := calculateModulesStates(filepath.Join("testdata/golden/updated-release", bufstate.SyncRoot), prevRelease, currentRelease)
		require.NoError(t, err)
		assert.Equal(t, map[string]releaseModuleState{
			"envoyproxy/envoy": {
				status: modules.Updated,
				references: []*statev1alpha1.ModuleReference{{
					Name:   "7850b6bb6494e3bfc093b1aff20282ab30b67940",
					Digest: "updatedDummyManifestDigestEnvoy",
				}},
			}}, got)
		assert.True(t, newContent)
	})

	t.Run("NoChangesRelease-FromPreviousRelease", func(t *testing.T) {
		t.Parallel()
		prevRelease := map[string]string{
			"envoyproxy/envoy": "7850b6bb6494e3bfc093b1aff20282ab30b67940",
		}
		currentRelease := map[string]string{
			"envoyproxy/envoy": "7850b6bb6494e3bfc093b1aff20282ab30b67940",
		}
		got, newContent, err := calculateModulesStates(filepath.Join("not-relevant", bufstate.SyncRoot), prevRelease, currentRelease)
		require.NoError(t, err)
		assert.Equal(t, map[string]releaseModuleState{
			"envoyproxy/envoy": {
				status: modules.Unchanged,
			}}, got)
		assert.False(t, newContent)
	})

	t.Run("NewUpdatedAndUnchanged-FromPreviousRelease", func(t *testing.T) {
		t.Parallel()
		prevRelease := map[string]string{
			"envoyproxy/envoy":               "bb554f53ad8d3a2a2ae4cbd7102a3e20ae00b558",
			"envoyproxy/protoc-gen-validate": "38260ee45796b420276ac925d826ecec8fc3e9a8",
		}
		currentRelease := map[string]string{
			"envoyproxy/envoy":               "7850b6bb6494e3bfc093b1aff20282ab30b67940", // updated
			"envoyproxy/protoc-gen-validate": "38260ee45796b420276ac925d826ecec8fc3e9a8", // unchanged
			"gogo/protobuf":                  "8892e00f944642b7dc8d81b419879fd4be12f056", // new
		}
		got, newContent, err := calculateModulesStates(filepath.Join("testdata/golden/newupdatedandunchanged-release", bufstate.SyncRoot), prevRelease, currentRelease)
		require.NoError(t, err)
		assert.Equal(t, map[string]releaseModuleState{
			"envoyproxy/protoc-gen-validate": {
				status: modules.Unchanged,
			},
			"envoyproxy/envoy": {
				status: modules.Updated,
				references: []*statev1alpha1.ModuleReference{{
					Name:   "7850b6bb6494e3bfc093b1aff20282ab30b67940",
					Digest: "updatedDummyManifestDigestEnvoy",
				}},
			},
			"gogo/protobuf": {
				status: modules.New,
				references: []*statev1alpha1.ModuleReference{{
					Name:   "8892e00f944642b7dc8d81b419879fd4be12f056",
					Digest: "newDummyManifestDigestGogoProtobuf",
				}},
			}}, got)
		assert.True(t, newContent)
	})
}

func TestMapGlobalStateReferences(t *testing.T) {
	t.Parallel()
	t.Run("nil_state", func(t *testing.T) {
		t.Parallel()

		var globalState *statev1alpha1.GlobalState
		got := mapGlobalStateReferences(globalState)
		require.Equal(t, map[string]string(nil), got)
	})
	t.Run("not_nil_state", func(t *testing.T) {
		t.Parallel()

		globalState := &statev1alpha1.GlobalState{
			Modules: []*statev1alpha1.GlobalStateReference{
				{
					ModuleName:      "test-org/test-repo",
					LatestReference: "v1.0.0",
				}, {
					ModuleName:      "other-org/other-repo",
					LatestReference: "v1.0.0",
				},
			},
		}
		got := mapGlobalStateReferences(globalState)
		require.Equal(t, map[string]string{
			"test-org/test-repo":   "v1.0.0",
			"other-org/other-repo": "v1.0.0",
		}, got)
	})
}

func TestCreateReleaseBody(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()
		mods := map[string]releaseModuleState{
			"test-org/test-repo": {
				status: modules.New,
				references: []*statev1alpha1.ModuleReference{
					{
						Name:   "v1.0.0",
						Digest: "fakedigest",
					}, {
						Name:   "v1.1.0",
						Digest: "fakedigest",
					},
				},
			},
		}

		const want = `# Buf Modules Release 20230519.1

## New Modules
<details><summary>test-org/test-repo: 2 update(s)</summary>

| Reference | Manifest Digest |
|---|---|
| v1.0.0 | fakedigest |
| v1.1.0 | fakedigest |

</details>

`
		got, err := createReleaseBody("20230519.1", mods)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
	t.Run("NewAndUpdated", func(t *testing.T) {
		t.Parallel()
		mods := map[string]releaseModuleState{
			"test-org/new-repo": {
				status: modules.New,
				references: []*statev1alpha1.ModuleReference{
					{
						Name:   "v1.0.0",
						Digest: "fakedigest",
					}, {
						Name:   "v1.1.0",
						Digest: "fakedigest",
					},
				},
			},
			"test-org/updated-repo": {
				status: modules.Updated,
				references: []*statev1alpha1.ModuleReference{
					{
						Name:   "v1.0.0",
						Digest: "fakedigest",
					}, {
						Name:   "v1.1.0",
						Digest: "fakedigest",
					},
				},
			},
		}

		const want = `# Buf Modules Release 20230519.1

## New Modules
<details><summary>test-org/new-repo: 2 update(s)</summary>

| Reference | Manifest Digest |
|---|---|
| v1.0.0 | fakedigest |
| v1.1.0 | fakedigest |

</details>

## Updated Modules
<details><summary>test-org/updated-repo: 2 update(s)</summary>

| Reference | Manifest Digest |
|---|---|
| v1.0.0 | fakedigest |
| v1.1.0 | fakedigest |

</details>

`
		got, err := createReleaseBody("20230519.1", mods)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
	t.Run("NewUpdatedUnchanged", func(t *testing.T) {
		t.Parallel()
		mods := map[string]releaseModuleState{
			"test-org/new-repo": {
				status: modules.New,
				references: []*statev1alpha1.ModuleReference{
					{
						Name:   "v1.0.0",
						Digest: "fakedigest",
					},
				},
			},
			"test-org/updated-repo": {
				status: modules.Updated,
				references: []*statev1alpha1.ModuleReference{
					{
						Name:   "v1.0.0",
						Digest: "fakedigest",
					},
				},
			},
			"test-org/unchanged-repo": {
				status: modules.Unchanged,
				references: []*statev1alpha1.ModuleReference{
					{
						Name:   "v1.0.0",
						Digest: "fakedigest",
					},
				},
			},
		}

		const want = `# Buf Modules Release 20230519.1

## New Modules
<details><summary>test-org/new-repo: 1 update(s)</summary>

| Reference | Manifest Digest |
|---|---|
| v1.0.0 | fakedigest |

</details>

## Updated Modules
<details><summary>test-org/updated-repo: 1 update(s)</summary>

| Reference | Manifest Digest |
|---|---|
| v1.0.0 | fakedigest |

</details>

## Unchanged Modules
<details><summary>Expand</summary>

- test-org/unchanged-repo

</details>
`
		got, err := createReleaseBody("20230519.1", mods)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

func TestWriteReferencesTable(t *testing.T) {
	t.Parallel()

	populateReferences := func(refsCount int) []*statev1alpha1.ModuleReference {
		refs := make([]*statev1alpha1.ModuleReference, refsCount)
		for i := 0; i < refsCount; i++ {
			refs[i] = &statev1alpha1.ModuleReference{
				Name:   fmt.Sprintf("commit%03d", i),
				Digest: fmt.Sprintf("digest%03d", i),
			}
		}
		return refs
	}
	t.Run("lessThanLimit", func(t *testing.T) {
		t.Parallel()

		const (
			moduleName = "foo/bar"
			refsCount  = 5
		)
		var strBuilder strings.Builder
		require.NoError(t, writeUpdatedReferencesTable(&strBuilder, moduleName, populateReferences(refsCount)))
		const want = `<details><summary>foo/bar: 5 update(s)</summary>

| Reference | Manifest Digest |
|---|---|
| commit000 | digest000 |
| commit001 | digest001 |
| commit002 | digest002 |
| commit003 | digest003 |
| commit004 | digest004 |

</details>
`
		assert.Equal(t, want, strBuilder.String())
	})
	t.Run("rightInTheLimit", func(t *testing.T) {
		t.Parallel()

		const (
			moduleName = "foo/bar"
			refsCount  = 10
		)
		var strBuilder strings.Builder
		require.NoError(t, writeUpdatedReferencesTable(&strBuilder, moduleName, populateReferences(refsCount)))
		const want = `<details><summary>foo/bar: 10 update(s)</summary>

| Reference | Manifest Digest |
|---|---|
| commit000 | digest000 |
| commit001 | digest001 |
| commit002 | digest002 |
| commit003 | digest003 |
| commit004 | digest004 |
| commit005 | digest005 |
| commit006 | digest006 |
| commit007 | digest007 |
| commit008 | digest008 |
| commit009 | digest009 |

</details>
`
		assert.Equal(t, want, strBuilder.String())
	})
	t.Run("afterLimit", func(t *testing.T) {
		t.Parallel()

		const (
			moduleName = "foo/bar"
			refsCount  = 100
		)
		var strBuilder strings.Builder
		require.NoError(t, writeUpdatedReferencesTable(&strBuilder, moduleName, populateReferences(refsCount)))
		const want = `<details><summary>foo/bar: 100 update(s)</summary>

| Reference | Manifest Digest |
|---|---|
| commit000 | digest000 |
| commit001 | digest001 |
| commit002 | digest002 |
| commit003 | digest003 |
| commit004 | digest004 |
| ... 90 references skipped ... | ... 90 references skipped ... |
| commit095 | digest095 |
| commit096 | digest096 |
| commit097 | digest097 |
| commit098 | digest098 |
| commit099 | digest099 |

</details>
`
		assert.Equal(t, want, strBuilder.String())
	})
}
