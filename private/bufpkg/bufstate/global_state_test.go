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

func TestValidGlobalStates(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(&statev1alpha1.GlobalState{}))
	})
	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(&statev1alpha1.GlobalState{
			Modules: []*statev1alpha1.GlobalStateReference{
				{ModuleName: "aaa/bbb", LatestReference: "foo"},
				{ModuleName: "ccc/ddd", LatestReference: "bar"},
			},
		}))
	})
	t.Run("repeatedReferencesForDifferentModules", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(&statev1alpha1.GlobalState{
			Modules: []*statev1alpha1.GlobalStateReference{
				{ModuleName: "aaa/bbb", LatestReference: "foo"},
				{ModuleName: "ccc/ddd", LatestReference: "foo"},
			},
		}))
	})
}

func TestInvalidGlobalStates(t *testing.T) {
	t.Parallel()
	t.Run("repeatedModules", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validate(&statev1alpha1.GlobalState{
			Modules: []*statev1alpha1.GlobalStateReference{
				{ModuleName: "aaa/bbb", LatestReference: "foo"},
				{ModuleName: "aaa/bbb", LatestReference: "bar"},
				{ModuleName: "aaa/ccc", LatestReference: "baz"},
			},
		}))
	})
	t.Run("emptyReferences", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validate(&statev1alpha1.GlobalState{
			Modules: []*statev1alpha1.GlobalStateReference{
				{ModuleName: "aaa/bbb", LatestReference: "foo"},
				{ModuleName: "aaa/ccc", LatestReference: ""},
				{ModuleName: "aaa/ddd", LatestReference: "bar"},
			},
		}))
	})
	t.Run("emptyModuleNames", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validate(&statev1alpha1.GlobalState{
			Modules: []*statev1alpha1.GlobalStateReference{
				{ModuleName: "aaa/bbb", LatestReference: "foo"},
				{ModuleName: "", LatestReference: "foo"},
				{ModuleName: "aaa/ddd", LatestReference: "bar"},
			},
		}))
	})
}
