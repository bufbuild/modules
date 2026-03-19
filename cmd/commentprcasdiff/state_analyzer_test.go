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
	"testing"

	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAppendedRefs(t *testing.T) {
	t.Parallel()
	ref := func(name, digest string) *statev1alpha1.ModuleReference {
		return statev1alpha1.ModuleReference_builder{Name: name, Digest: digest}.Build()
	}
	type testCase struct {
		name         string
		baseRefs     []*statev1alpha1.ModuleReference
		headRefs     []*statev1alpha1.ModuleReference
		wantCurrent  *statev1alpha1.ModuleReference
		wantAppended []*statev1alpha1.ModuleReference
	}
	testCases := []testCase{
		{
			// base=0, head=0 → nothing to do
			name:         "both_empty",
			baseRefs:     nil,
			headRefs:     nil,
			wantCurrent:  nil,
			wantAppended: nil,
		},
		{
			// base=0, head=1 → head[0] is baseline, nothing appended
			name:         "base_empty_single_ref_in_head",
			baseRefs:     nil,
			headRefs:     []*statev1alpha1.ModuleReference{ref("v1.0.0", "d1")},
			wantCurrent:  ref("v1.0.0", "d1"),
			wantAppended: []*statev1alpha1.ModuleReference{},
		},
		{
			// base=0, head=3 → head[0] is baseline, head[1:] are appended
			name:     "base_empty_multiple_refs",
			baseRefs: nil,
			headRefs: []*statev1alpha1.ModuleReference{
				ref("v1.0.0", "d1"),
				ref("v2.0.0", "d2"),
				ref("v3.0.0", "d2"),
			},
			wantCurrent: ref("v1.0.0", "d1"),
			wantAppended: []*statev1alpha1.ModuleReference{
				ref("v2.0.0", "d2"),
				ref("v3.0.0", "d2"),
			},
		},
		{
			// base=2, head=4 → base[latest] is baseline, head[2:] are appended
			name: "base_non_empty_some_appends",
			baseRefs: []*statev1alpha1.ModuleReference{
				ref("v1.0.0", "d1"),
				ref("v1.1.0", "d1"),
			},
			headRefs: []*statev1alpha1.ModuleReference{
				ref("v1.0.0", "d1"),
				ref("v1.1.0", "d1"),
				ref("v2.0.0", "d2"),
				ref("v2.1.0", "d2"),
			},
			wantCurrent: ref("v1.1.0", "d1"),
			wantAppended: []*statev1alpha1.ModuleReference{
				ref("v2.0.0", "d2"),
				ref("v2.1.0", "d2"),
			},
		},
		{
			// base=1, head=1 → base[latest] is baseline, no appends
			name:         "head_count_equals_base_count_not_supported",
			baseRefs:     []*statev1alpha1.ModuleReference{ref("v1.0.0", "d1")},
			headRefs:     []*statev1alpha1.ModuleReference{ref("v1.0.0", "d1")},
			wantCurrent:  ref("v1.0.0", "d1"),
			wantAppended: nil,
		},
		{
			// base=2, head=1 → base[latest] is baseline, no appends
			name: "head_shorter_than_base_not_supported",
			baseRefs: []*statev1alpha1.ModuleReference{
				ref("v1.0.0", "d1"),
				ref("v2.0.0", "d2"),
			},
			headRefs:     []*statev1alpha1.ModuleReference{ref("v1.0.0", "d1")},
			wantCurrent:  ref("v2.0.0", "d2"),
			wantAppended: nil,
		},
		{
			// base=3, head=5, but the digests don't match. The function only looks at counts, not
			// content: base[latest] is baseline, head[2:] are appended.
			name: "existing_ref_modified_not_supported",
			baseRefs: []*statev1alpha1.ModuleReference{
				ref("v1.0.0", "d1"),
				ref("v2.0.0", "d2"),
				ref("v3.0.0", "d3"),
			},
			headRefs: []*statev1alpha1.ModuleReference{
				ref("v1.0.0", "d1-modified"),
				ref("v2.0.0", "d2-modified"),
				ref("v3.0.0", "d3-modified"),
				ref("v4.0.0", "d4"),
				ref("v5.0.0", "d5"),
			},
			wantCurrent: ref("v3.0.0", "d3"), // the one from base, not from head
			wantAppended: []*statev1alpha1.ModuleReference{
				ref("v4.0.0", "d4"),
				ref("v5.0.0", "d5"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotCurrent, gotAppended := resolveAppendedRefs(tc.baseRefs, tc.headRefs)
			assert.Equal(t, tc.wantCurrent, gotCurrent)
			assert.Equal(t, tc.wantAppended, gotAppended)
		})
	}
}

func TestParseLineNumbersFromDiff(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name          string
		diffOutput    string
		expectedCount int
		want          []int
	}
	testCases := []testCase{
		{
			name: "single_digest_change",
			diffOutput: `@@ -275,0 +276,6 @@
+    },
+    {
+      "name": "v1.1.0",
+      "digest": "aaa"
+    }
+  ]`,
			expectedCount: 1,
			want:          []int{279},
		},
		{
			name: "multiple_digest_changes",
			diffOutput: `@@ -100,0 +101,12 @@
+    },
+    {
+      "name": "v1.1.0",
+      "digest": "aaa"
+    },
+    {
+      "name": "v1.2.0",
+      "digest": "bbb"
+    },
+    {
+      "name": "v1.3.0",
+      "digest": "ccc"`,
			expectedCount: 3,
			want:          []int{104, 108, 112},
		},
		{
			name: "multiple_hunks",
			diffOutput: `@@ -50,0 +51,4 @@
+    {
+      "name": "v2.0.0",
+      "digest": "xxx"
+    }
@@ -100,0 +105,8 @@
+    {
+      "name": "v3.0.0",
+      "digest": "yyy"
+    },
+    {
+      "name": "v4.0.0",
+      "digest": "zzz"
+    }`,
			expectedCount: 3,
			want:          []int{53, 107, 111},
		},
		{
			name: "no_digest_lines",
			diffOutput: `@@ -10,0 +11,4 @@
+    {
+      "name": "v1.0.0",
+      "other": "field"
+    }`,
			expectedCount: 1,
			want:          []int{0},
		},
		{
			name:          "empty_diff",
			diffOutput:    "",
			expectedCount: 1,
			want:          []int{0},
		},
		{
			name: "digest_with_formatting",
			diffOutput: `@@ -200,0 +201,8 @@
+{
+"name": "v0.1.0",
+"digest": "abc123"
+},
+		{
+			"name": "v0.2.0",
+			"digest": "def456"
+		}`,
			expectedCount: 2,
			want:          []int{203, 207},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseLineNumbersFromDiff(tc.diffOutput, tc.expectedCount)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
