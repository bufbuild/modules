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
)

// ChangedModuleStatesFromPR returns module state directories that changed in the PR.
// Returns paths like "modules/sync/bufbuild/protovalidate" (without /state.json suffix).
func ChangedModuleStatesFromPR(
	ctx context.Context,
	prNumber string,
	baseRef string,
	headRef string,
) ([]string, error) {
	// Get list of changed files in the PR
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", baseRef, headRef)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	// Filter for state.json files in modules/sync/
	var modulePaths []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for state.json files in modules/sync/
		if strings.HasPrefix(line, "modules/sync/") && strings.HasSuffix(line, "/state.json") {
			// Extract module path (remove /state.json suffix)
			modulePath := filepath.Dir(line)
			if !seen[modulePath] {
				seen[modulePath] = true
				modulePaths = append(modulePaths, modulePath)
			}
		}
	}

	return modulePaths, nil
}
