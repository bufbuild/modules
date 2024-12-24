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

package bufstate

import (
	"testing"

	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestValidModuleStates(t *testing.T) {
	t.Parallel()
	readWriter, err := NewReadWriter()
	require.NoError(t, err)
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, readWriter.validator.Validate(&statev1alpha1.ModuleState{}))
	})
	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, readWriter.validator.Validate(statev1alpha1.ModuleState_builder{
			References: []*statev1alpha1.ModuleReference{
				statev1alpha1.ModuleReference_builder{Name: "commit1", Digest: "foo"}.Build(),
				statev1alpha1.ModuleReference_builder{Name: "commit2", Digest: "bar"}.Build(),
				statev1alpha1.ModuleReference_builder{Name: "v1.23.4", Digest: "baz"}.Build(),
			},
		}.Build()))
	})
	t.Run("repeatedDigestsForDifferentReferences", func(t *testing.T) {
		// this happens all the time, many references with the same digest, meaning
		// between those commits there were no changes in the relevant files that we
		// sync.
		t.Parallel()
		require.NoError(t, readWriter.validator.Validate(statev1alpha1.ModuleState_builder{
			References: []*statev1alpha1.ModuleReference{
				statev1alpha1.ModuleReference_builder{Name: "commit1", Digest: "foo"}.Build(),
				statev1alpha1.ModuleReference_builder{Name: "commit2", Digest: "foo"}.Build(),
				statev1alpha1.ModuleReference_builder{Name: "v1.23.4", Digest: "foo"}.Build(),
			},
		}.Build()))
	})
}

func TestInvalidModuleStates(t *testing.T) {
	t.Parallel()
	readWriter, err := NewReadWriter()
	require.NoError(t, err)
	t.Run("repeatedReferences", func(t *testing.T) {
		t.Parallel()
		err := readWriter.validator.Validate(statev1alpha1.ModuleState_builder{
			References: []*statev1alpha1.ModuleReference{
				statev1alpha1.ModuleReference_builder{Name: "commit1", Digest: "foo"}.Build(),
				statev1alpha1.ModuleReference_builder{Name: "commit1", Digest: "bar"}.Build(),
				statev1alpha1.ModuleReference_builder{Name: "commit2", Digest: "baz"}.Build(),
			},
		}.Build())
		require.Contains(t, err.Error(), "reference commit1 has appeared multiple times")
	})
	t.Run("emptyDigests", func(t *testing.T) {
		// even if the reference has no files or empty content, an empty manifest
		// still has a digest.
		t.Parallel()
		err := readWriter.validator.Validate(statev1alpha1.ModuleState_builder{
			References: []*statev1alpha1.ModuleReference{
				statev1alpha1.ModuleReference_builder{Name: "commit1", Digest: "foo"}.Build(),
				statev1alpha1.ModuleReference_builder{Name: "commit2", Digest: ""}.Build(),
				statev1alpha1.ModuleReference_builder{Name: "commit3", Digest: "baz"}.Build(),
			},
		}.Build())
		require.Contains(t, err.Error(), "references[1].digest: value is required")
	})
	t.Run("emptyReferenceNames", func(t *testing.T) {
		// all commits should have a valid, unique reference
		t.Parallel()
		err := readWriter.validator.Validate(statev1alpha1.ModuleState_builder{
			References: []*statev1alpha1.ModuleReference{
				statev1alpha1.ModuleReference_builder{Name: "commit1", Digest: "foo"}.Build(),
				statev1alpha1.ModuleReference_builder{Name: "", Digest: "foo"}.Build(),
				statev1alpha1.ModuleReference_builder{Name: "commit3", Digest: "baz"}.Build(),
			},
		}.Build())
		require.Contains(t, err.Error(), "references[1].name: value is required")
	})
}
