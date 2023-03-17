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

package semverutil_test

import (
	"testing"

	"github.com/bufbuild/modules/internal/semverutil"
	"github.com/stretchr/testify/assert"
)

func TestSemverTagNamesAtLeast(t *testing.T) {
	t.Parallel()
	t.Run("inclusive", func(t *testing.T) {
		t.Parallel()
		got := semverutil.SemverTagNamesAtLeast([]string{
			"v1.0.0",
			"v2.0.0",
			"v2.1.0",
			"v3.0.0",
		}, "v2.0.0",
			true, // inclusive
		)
		assert.Equal(t, []string{
			"v2.0.0",
			"v2.1.0",
			"v3.0.0",
		}, got)
	})
	t.Run("exclusive", func(t *testing.T) {
		t.Parallel()
		got := semverutil.SemverTagNamesAtLeast([]string{
			"v1.0.0",
			"v2.0.0",
			"v2.1.0",
			"v3.0.0",
		}, "v2.0.0",
			false, // exclusive
		)
		assert.Equal(t, []string{
			// "v2.0.0", // excluded
			"v2.1.0",
			"v3.0.0",
		}, got)
	})
}
