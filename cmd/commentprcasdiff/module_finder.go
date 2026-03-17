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
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"buf.build/go/standard/xslices"
)

// changedModuleStates returns module directories that had *any change* on its state.json files
// between the passed base and head refs. Returns paths like "modules/sync/bufbuild/protovalidate"
// (without /state.json suffix).
func changedModuleStates(
	ctx context.Context,
	baseRef string,
	headRef string,
) ([]string, error) {
	// Get list of changed files in the PR
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", baseRef, headRef) //nolint:gosec
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	// Filter for state.json files in modules/sync/
	modulePaths := make(map[string]struct{})
	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Look for "modules/sync/<owner>/<module>/state.json" files
		if strings.HasPrefix(line, "modules/sync/") && strings.HasSuffix(line, "/state.json") {
			// Extract module path (remove /state.json suffix)
			modulePath := filepath.Dir(line)
			modulePaths[modulePath] = struct{}{}
		}
	}

	return xslices.MapKeysToSortedSlice(modulePaths), nil
}
