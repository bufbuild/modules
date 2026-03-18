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
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"buf.build/go/standard/xslices"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var dryRun bool
	flag.BoolVar(&dryRun, "dry-run", false, "print comments to stdout instead of posting to GitHub")
	flag.Parse()

	// Read environment variables
	prNumber := os.Getenv("PR_NUMBER")
	baseRef := os.Getenv("BASE_REF")
	headRef := os.Getenv("HEAD_REF")

	if prNumber == "" && !dryRun {
		return errors.New("PR_NUMBER environment variable is required when not a dry-run")
	}
	if baseRef == "" {
		return errors.New("BASE_REF environment variable is required")
	}
	if headRef == "" {
		return errors.New("HEAD_REF environment variable is required")
	}

	fmt.Fprintf(
		os.Stdout,
		"Processing PR #%s (base: %s, head: %s)\n",
		cmp.Or(prNumber, "dry-run"),
		baseRef,
		headRef,
	)

	// Find changed module state directories
	moduleStatePaths, err := changedModuleStateFiles(ctx, baseRef, headRef)
	if err != nil {
		return fmt.Errorf("find changed modules: %w", err)
	}

	if len(moduleStatePaths) == 0 {
		fmt.Fprintf(
			os.Stdout,
			"No module state.json files changed in between %s..%s refs\n",
			baseRef,
			headRef,
		)
		return nil
	}
	moduleStatePathsSorted := xslices.MapKeysToSortedSlice(moduleStatePaths)

	fmt.Fprintf(
		os.Stdout,
		"Found %d changed module(s):\n%s\n",
		len(moduleStatePaths),
		strings.Join(moduleStatePathsSorted, "\n"),
	)

	// Collect all transitions from all modules
	var allTransitions []stateTransition
	for _, moduleStatePath := range moduleStatePathsSorted {
		fmt.Fprintf(os.Stdout, "Analyzing %s...\n", moduleStatePath)

		transitions, err := getStateFileTransitions(ctx, moduleStatePath, baseRef, headRef)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to analyze %s: %v\n", moduleStatePath, err)
			continue
		}

		if len(transitions) > 0 {
			fmt.Fprintf(os.Stdout, "  Found %d digest transition(s)\n", len(transitions))
			allTransitions = append(allTransitions, transitions...)
		} else {
			fmt.Fprintf(os.Stdout, "  No digest changes\n")
		}
	}

	if len(allTransitions) == 0 {
		fmt.Fprintf(os.Stdout, "No digest transitions found\n")
		return nil
	}

	fmt.Fprintf(os.Stdout, "\nRunning casdiff for %d transition(s)...\n", len(allTransitions))
	results := runCASDiffs(ctx, allTransitions)

	// Separate successful results from errors
	var (
		successfulResults []casDiffResult
		failedResults     []casDiffResult
	)

	for _, result := range results {
		if result.err != nil {
			failedResults = append(failedResults, result)
			fmt.Fprintf(
				os.Stderr, "Warning: casdiff failed for %s %s->%s: %v\n",
				result.transition.modulePath,
				result.transition.fromRef,
				result.transition.toRef,
				result.err,
			)
		} else {
			successfulResults = append(successfulResults, result)
		}
	}

	// Post or print comments for successful results
	var postErrs []error
	if len(successfulResults) > 0 {
		if dryRun {
			fmt.Fprintf(os.Stdout, "\n[dry-run] %d comment(s) would be posted:\n", len(successfulResults))
			for _, result := range successfulResults {
				fmt.Fprintf(os.Stdout, "\n--- %s (line %d: %s -> %s) ---\n%s\n",
					result.transition.filePath,
					result.transition.lineNumber,
					result.transition.fromRef,
					result.transition.toRef,
					result.output,
				)
			}
		} else {
			fmt.Fprintf(os.Stdout, "\nPosting %d comment(s) to PR...\n", len(successfulResults))

			comments := make([]prReviewComment, len(successfulResults))
			for i, result := range successfulResults {
				comments[i] = prReviewComment{
					prNumber:   prNumber,
					filePath:   result.transition.filePath,
					lineNumber: result.transition.lineNumber,
					body:       result.output,
					commitID:   headRef,
				}
			}

			postErrs = postReviewComments(ctx, comments...)
			posted := 0
			for _, err := range postErrs {
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: failed to post comment: %v\n", err)
				} else {
					posted++
				}
			}
			fmt.Fprintf(os.Stdout, "Successfully posted %d comment(s)\n", posted)
		}
	}

	// Report failures and exit non-zero if anything went wrong.
	if len(failedResults) > 0 {
		fmt.Fprintf(os.Stderr, "\nSummary: %d casdiff command(s) failed:\n", len(failedResults))
		for _, result := range failedResults {
			fmt.Fprintf(os.Stderr, "  - %s: %s -> %s\n",
				result.transition.modulePath,
				result.transition.fromRef,
				result.transition.toRef)
		}
		return fmt.Errorf("%d casdiff command(s) failed", len(failedResults))
	}
	for _, err := range postErrs {
		if err != nil {
			return fmt.Errorf("one or more comments failed to post")
		}
	}

	fmt.Fprintf(os.Stdout, "\nDone!\n")
	return nil
}
