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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLineNumbersFromDiff(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name          string
		diffOutput    string
		expectedCount int
		want          []int
		wantErr       bool
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
