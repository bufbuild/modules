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
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// Read environment variables
	prNumber := os.Getenv("PR_NUMBER")
	baseRef := os.Getenv("BASE_REF")
	headRef := os.Getenv("HEAD_REF")

	if prNumber == "" {
		return fmt.Errorf("PR_NUMBER environment variable is required")
	}
	if baseRef == "" {
		return fmt.Errorf("BASE_REF environment variable is required")
	}
	if headRef == "" {
		return fmt.Errorf("HEAD_REF environment variable is required")
	}

	fmt.Printf("Processing PR #%s (base: %s, head: %s)\n", prNumber, baseRef, headRef)

	// Find changed module state directories
	modulePaths, err := ChangedModuleStatesFromPR(ctx, prNumber, baseRef, headRef)
	if err != nil {
		return fmt.Errorf("find changed modules: %w", err)
	}

	if len(modulePaths) == 0 {
		fmt.Println("No module state.json files changed in this PR")
		return nil
	}

	fmt.Printf("Found %d changed module(s)\n", len(modulePaths))

	// Collect all transitions from all modules
	var allTransitions []StateTransition
	for _, modulePath := range modulePaths {
		stateFilePath := modulePath + "/state.json"
		fmt.Printf("Analyzing %s...\n", stateFilePath)

		transitions, err := GetStateFileTransitions(ctx, stateFilePath, baseRef, headRef)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to analyze %s: %v\n", stateFilePath, err)
			continue
		}

		if len(transitions) > 0 {
			fmt.Printf("  Found %d digest transition(s)\n", len(transitions))
			allTransitions = append(allTransitions, transitions...)
		} else {
			fmt.Printf("  No digest changes\n")
		}
	}

	if len(allTransitions) == 0 {
		fmt.Println("No digest transitions found")
		return nil
	}

	fmt.Printf("\nRunning casdiff for %d transition(s)...\n", len(allTransitions))

	// Run casdiff for all transitions in parallel
	results := RunCASDiffParallel(ctx, allTransitions)

	// Separate successful results from errors
	var successfulResults []CASDiffResult
	var failedResults []CASDiffResult

	for _, result := range results {
		if result.Error != nil {
			failedResults = append(failedResults, result)
			fmt.Fprintf(os.Stderr, "Warning: casdiff failed for %s %s->%s: %v\n",
				result.Transition.ModulePath,
				result.Transition.FromRef,
				result.Transition.ToRef,
				result.Error)
		} else {
			successfulResults = append(successfulResults, result)
		}
	}

	// Post comments for successful results
	if len(successfulResults) > 0 {
		fmt.Printf("\nPosting %d comment(s) to PR...\n", len(successfulResults))

		comments := make([]PRReviewComment, len(successfulResults))
		for i, result := range successfulResults {
			comments[i] = PRReviewComment{
				PRNumber:   prNumber,
				FilePath:   result.Transition.FilePath,
				LineNumber: result.Transition.LineNumber,
				Body:       result.Output,
				CommitID:   headRef,
			}
		}

		errors := PostReviewComments(ctx, comments...)
		for _, err := range errors {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to post comment: %v\n", err)
			}
		}

		fmt.Printf("Successfully posted %d comment(s)\n", len(successfulResults)-len(errors))
	}

	// Log summary of failures
	if len(failedResults) > 0 {
		fmt.Fprintf(os.Stderr, "\nSummary: %d casdiff command(s) failed:\n", len(failedResults))
		for _, result := range failedResults {
			fmt.Fprintf(os.Stderr, "  - %s: %s -> %s\n",
				result.Transition.ModulePath,
				result.Transition.FromRef,
				result.Transition.ToRef)
		}
	}

	fmt.Println("\nDone!")
	return nil
}
