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
	"reflect"
	"testing"
)

func TestParseLineNumbersFromDiff(t *testing.T) {
	tests := []struct {
		name          string
		diffOutput    string
		expectedCount int
		want          []int
		wantErr       bool
	}{
		{
			name: "single_digest_change",
			diffOutput: `@@ -275,0 +276,6 @@
+    },
+    {
+      "name": "v1.1.0",
+      "digest": "35b3d88f6b0fbf159d9eedfc2fbfa976490e6bca1d98914c1c71f29bfe2da6261fca56c057975b3e5c022b3234a0f2eea8e2d1b599a937c6c5d63d21201a9bc3"
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
@@ -100,0 +105,4 @@
+    {
+      "name": "v3.0.0",
+      "digest": "yyy"
+    }`,
			expectedCount: 2,
			want:          []int{53, 107},
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
			name: "digest_with_spaces_and_formatting",
			diffOutput: `@@ -200,0 +201,8 @@
+    {
+      "name": "v0.1.0",
+      "digest": "abc123"
+    },
+    {
+      "name": "v0.2.0",
+      "digest": "def456"
+    }`,
			expectedCount: 2,
			want:          []int{203, 207},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLineNumbersFromDiff(tt.diffOutput, tt.expectedCount)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLineNumbersFromDiff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLineNumbersFromDiff() = %v, want %v", got, tt.want)
			}
		})
	}
}
