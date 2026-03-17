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
	"os/exec"
	"sync"
)

// CASDiffResult contains the result of running casdiff for a transition.
type CASDiffResult struct {
	Transition StateTransition
	Output     string // Markdown output from casdiff
	Error      error
}

// RunCASDiff executes casdiff command in the module directory.
func RunCASDiff(ctx context.Context, transition StateTransition) CASDiffResult {
	result := CASDiffResult{
		Transition: transition,
	}

	// Run casdiff in the module directory
	cmd := exec.CommandContext(
		ctx,
		"go", "run", "../../cmd/casdiff",
		transition.FromRef,
		transition.ToRef,
		"--format=markdown",
	)
	cmd.Dir = transition.ModulePath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result.Error = fmt.Errorf("casdiff failed: %w (stderr: %s)", err, stderr.String())
		return result
	}

	result.Output = stdout.String()
	return result
}

// RunCASDiffParallel runs multiple casdiff commands concurrently.
func RunCASDiffParallel(ctx context.Context, transitions []StateTransition) []CASDiffResult {
	results := make([]CASDiffResult, len(transitions))
	var wg sync.WaitGroup

	for i, transition := range transitions {
		wg.Add(1)
		go func(index int, trans StateTransition) {
			defer wg.Done()
			results[index] = RunCASDiff(ctx, trans)
		}(i, transition)
	}

	wg.Wait()
	return results
}
