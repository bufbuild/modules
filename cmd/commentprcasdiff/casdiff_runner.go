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
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// casDiffResult contains the result of running casdiff for a transition.
type casDiffResult struct {
	transition stateTransition
	output     string // Markdown output from casdiff
	err        error
}

// runCASDiff executes casdiff command in the module directory.
func runCASDiff(ctx context.Context, transition stateTransition) casDiffResult {
	result := casDiffResult{
		transition: transition,
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		result.err = fmt.Errorf("get working directory: %w", err)
		return result
	}

	// Run casdiff in the module directory. casdiff reads state.json from "." so it must run from the
	// module directory. We use an absolute path to the package to avoid path resolution issues when
	// cmd.Dir is set.
	cmd := exec.CommandContext( //nolint:gosec
		ctx,
		"go", "run", filepath.Join(repoRoot, "cmd", "casdiff"),
		transition.fromRef,
		transition.toRef,
		"--format=markdown",
	)
	cmd.Dir = filepath.Join(repoRoot, transition.modulePath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result.err = fmt.Errorf("casdiff failed: %w (stderr: %s)", err, stderr.String())
		return result
	}

	result.output = stdout.String()
	return result
}

// runCASDiffs runs multiple casdiff commands concurrently.
func runCASDiffs(ctx context.Context, transitions []stateTransition) []casDiffResult {
	results := make([]casDiffResult, len(transitions))
	var wg sync.WaitGroup

	for i, transition := range transitions {
		wg.Add(1)
		go func(index int, trans stateTransition) {
			defer wg.Done()
			results[index] = runCASDiff(ctx, trans)
		}(i, transition)
	}

	wg.Wait()
	return results
}
