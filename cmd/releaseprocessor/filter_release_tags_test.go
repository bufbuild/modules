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
)

func TestFilterReleaseTags(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name      string
		ownerName string
		repoName  string
		inclusive bool
		startRef  string
		tags      []string
		want      []string
	}
	testCases := []testCase{
		{
			name:      "skipped",
			ownerName: "protocolbuffers",
			repoName:  "protobuf",
			tags: []string{
				"v3.4.0",
				"v3.4.1",
				"v3.4.2",
			},
			want: []string{
				"v3.4.0",
				// "v3.4.1", // mark as ignored
				"v3.4.2",
			},
		},
		{
			name:      "inclusive_no_ref",
			inclusive: true,
			startRef:  "",
			tags: []string{
				"v1.2.2",
				"v1.2.3",
				"v1.2.4",
			},
			want: []string{
				"v1.2.2",
				"v1.2.3",
				"v1.2.4",
			},
		},
		{
			name:      "exclusive_no_ref",
			inclusive: false,
			startRef:  "",
			tags: []string{
				"v1.2.2",
				"v1.2.3",
				"v1.2.4",
			},
			want: []string{
				"v1.2.2",
				"v1.2.3",
				"v1.2.4",
			},
		},
		{
			name:      "inclusive",
			inclusive: true,
			startRef:  "v1.2.3",
			tags: []string{
				"v1.2.2",
				"v1.2.3",
				"v1.2.4",
			},
			want: []string{
				"v1.2.3",
				"v1.2.4",
			},
		},
		{
			name:      "exclusive",
			inclusive: false,
			startRef:  "v1.2.3",
			tags: []string{
				"v1.2.2",
				"v1.2.3",
				"v1.2.4",
			},
			want: []string{
				"v1.2.4",
			},
		},
		{
			name: "sorted",
			tags: []string{
				"v1.2.2",
				"v1.2.3",
				"v1.2.4",
			},
			want: []string{
				"v1.2.2",
				"v1.2.3",
				"v1.2.4",
			},
		},
		{
			name: "unsorted",
			tags: []string{
				"v1.2.4",
				"v1.2.4-rc.111",
				"v1.2.4-rc.22",
				"v1.2.4-rc.3",
				"v1.2.3",
				"v1.2.2",
			},
			want: []string{
				"v1.2.2",
				"v1.2.3",
				// release candidates are sorted by number, not lexicographically
				"v1.2.4-rc.3",
				"v1.2.4-rc.22",
				"v1.2.4-rc.111",
				"v1.2.4",
			},
		},
		{
			name: "unstable_semvers",
			tags: []string{
				"v1.2.3-alpha1",
				"v1.2.3-beta1",
				"v1.2.3-rc1",
				"v1.2.3-rc.1+build",
				"v1.2.3-rc.111",
				"v1.2.3-rc.22",
				"v1.2.3-rc.3",
				"v1.2.3",
			},
			want: []string{
				"v1.2.3-rc.3",
				"v1.2.3-rc.22",
				"v1.2.3-rc.111",
				"v1.2.3",
			},
		},
	}
	for _, tc := range testCases { //nolint:varnamelen // tc is an standard varname for test cases
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := command{
				owner:     tc.ownerName,
				repo:      tc.repoName,
				inclusive: tc.inclusive,
				reference: tc.startRef,
			}
			assert.Equal(t, tc.want, cmd.filterReleaseTags(tc.tags))
		})
	}
}
