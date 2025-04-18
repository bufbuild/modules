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
	"testing"

	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestValidGlobalStates(t *testing.T) {
	t.Parallel()
	readWriter, err := NewReadWriter()
	require.NoError(t, err)
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, readWriter.validator.Validate(&statev1alpha1.GlobalState{}))
	})
	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, readWriter.validator.Validate(statev1alpha1.GlobalState_builder{
			Modules: []*statev1alpha1.GlobalStateReference{
				statev1alpha1.GlobalStateReference_builder{ModuleName: "aaa/bbb", LatestReference: "foo"}.Build(),
				statev1alpha1.GlobalStateReference_builder{ModuleName: "ccc/ddd", LatestReference: "bar"}.Build(),
			},
		}.Build()))
	})
	t.Run("repeatedReferencesForDifferentModules", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, readWriter.validator.Validate(statev1alpha1.GlobalState_builder{
			Modules: []*statev1alpha1.GlobalStateReference{
				statev1alpha1.GlobalStateReference_builder{ModuleName: "aaa/bbb", LatestReference: "foo"}.Build(),
				statev1alpha1.GlobalStateReference_builder{ModuleName: "ccc/ddd", LatestReference: "foo"}.Build(),
			},
		}.Build()))
	})
}

func TestInvalidGlobalStates(t *testing.T) {
	t.Parallel()
	readWriter, err := NewReadWriter()
	require.NoError(t, err)
	t.Run("repeatedModules", func(t *testing.T) {
		t.Parallel()
		err := readWriter.validator.Validate(statev1alpha1.GlobalState_builder{
			Modules: []*statev1alpha1.GlobalStateReference{
				statev1alpha1.GlobalStateReference_builder{ModuleName: "aaa/bbb", LatestReference: "foo"}.Build(),
				statev1alpha1.GlobalStateReference_builder{ModuleName: "aaa/bbb", LatestReference: "bar"}.Build(),
				statev1alpha1.GlobalStateReference_builder{ModuleName: "aaa/ccc", LatestReference: "baz"}.Build(),
			},
		}.Build())
		require.Contains(t, err.Error(), "module name aaa/bbb has appeared multiple times")
	})
	t.Run("emptyReferences", func(t *testing.T) {
		t.Parallel()
		err := readWriter.validator.Validate(statev1alpha1.GlobalState_builder{
			Modules: []*statev1alpha1.GlobalStateReference{
				statev1alpha1.GlobalStateReference_builder{ModuleName: "aaa/bbb", LatestReference: "foo"}.Build(),
				statev1alpha1.GlobalStateReference_builder{ModuleName: "aaa/ccc", LatestReference: ""}.Build(),
				statev1alpha1.GlobalStateReference_builder{ModuleName: "aaa/ddd", LatestReference: "bar"}.Build(),
			},
		}.Build())
		require.Contains(t, err.Error(), "modules[1].latest_reference: value is required")
	})
	t.Run("emptyModuleNames", func(t *testing.T) {
		t.Parallel()
		err := readWriter.validator.Validate(statev1alpha1.GlobalState_builder{
			Modules: []*statev1alpha1.GlobalStateReference{
				statev1alpha1.GlobalStateReference_builder{ModuleName: "aaa/bbb", LatestReference: "foo"}.Build(),
				statev1alpha1.GlobalStateReference_builder{ModuleName: "", LatestReference: "foo"}.Build(),
				statev1alpha1.GlobalStateReference_builder{ModuleName: "aaa/ddd", LatestReference: "bar"}.Build(),
			},
		}.Build())
		require.Contains(t, err.Error(), "modules[1].module_name: value is required")
	})
}
