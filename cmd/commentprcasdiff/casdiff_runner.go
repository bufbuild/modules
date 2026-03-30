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
	"os"
	"path/filepath"
	"sync"

	"github.com/bufbuild/modules/internal/bufcasdiff"
)

// casDiffResult contains the result of running casdiff for a transition.
type casDiffResult struct {
	transition stateTransition
	output     string // Markdown output from casdiff
	err        error
}

// runCASDiff computes the CAS diff for a transition and returns its markdown output.
func runCASDiff(ctx context.Context, transition stateTransition) casDiffResult {
	result := casDiffResult{transition: transition}

	repoRoot, err := os.Getwd()
	if err != nil {
		result.err = fmt.Errorf("get working directory: %w", err)
		return result
	}
	moduleDirPath := filepath.Join(repoRoot, transition.modulePath)
	mdiff, err := bufcasdiff.DiffModuleDirectory(ctx, moduleDirPath, transition.fromRef, transition.toRef)
	if err != nil {
		result.err = fmt.Errorf("calculate casdiff: %w", err)
		return result
	}

	result.output = fmt.Sprintf(
		"```sh\n$ casdiff %s %s --format=markdown\n```\n<details><summary>%s</summary>\n<p>\n%s\n</p>\n</details>",
		transition.fromRef,
		transition.toRef,
		mdiff.Summary(),
		mdiff.String(bufcasdiff.ManifestDiffOutputFormatMarkdown),
	)
	return result
}

// runCASDiffs runs multiple casdiff commands concurrently.
func runCASDiffs(ctx context.Context, transitions []stateTransition) []casDiffResult {
	results := make([]casDiffResult, len(transitions))
	var wg sync.WaitGroup

	for i, transition := range transitions {
		wg.Go(func() {
			results[i] = runCASDiff(ctx, transition)
		})
	}

	wg.Wait()
	return results
}
