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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// StateTransition represents a digest change in a module's state.json file.
type StateTransition struct {
	ModulePath string // e.g., "modules/sync/bufbuild/protovalidate"
	FilePath   string // e.g., "modules/sync/bufbuild/protovalidate/state.json"
	FromRef    string // Last reference with old digest (e.g., "v1.1.0")
	ToRef      string // First reference with new digest (e.g., "v1.2.0")
	FromDigest string // Old digest
	ToDigest   string // New digest
	LineNumber int    // Line in diff where new digest first appears
}

type moduleState struct {
	References []reference `json:"references"`
}

type reference struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

// GetStateFileTransitions reads state.json from base and head branches,
// compares the JSON arrays to find appended references, and detects digest transitions.
func GetStateFileTransitions(
	ctx context.Context,
	filePath string,
	baseRef string,
	headRef string,
) ([]StateTransition, error) {
	// Read state.json from both branches
	baseContent, err := readFileAtRef(ctx, filePath, baseRef)
	if err != nil {
		return nil, fmt.Errorf("read base state: %w", err)
	}

	headContent, err := readFileAtRef(ctx, filePath, headRef)
	if err != nil {
		return nil, fmt.Errorf("read head state: %w", err)
	}

	// Parse JSON
	var baseState, headState moduleState
	if err := json.Unmarshal(baseContent, &baseState); err != nil {
		return nil, fmt.Errorf("parse base state JSON: %w", err)
	}
	if err := json.Unmarshal(headContent, &headState); err != nil {
		return nil, fmt.Errorf("parse head state JSON: %w", err)
	}

	// Identify appended references
	baseCount := len(baseState.References)
	headCount := len(headState.References)

	if headCount <= baseCount {
		// No new references added
		return nil, nil
	}

	appendedRefs := headState.References[baseCount:]

	// Get the baseline digest (last reference in base state)
	var (
		currentDigest string
		currentRef    string
	)
	if baseCount > 0 {
		lastBaseRef := baseState.References[baseCount-1]
		currentDigest = lastBaseRef.Digest
		currentRef = lastBaseRef.Name
	} else {
		// If base state is empty, use first appended ref as baseline
		if len(appendedRefs) > 0 {
			currentDigest = appendedRefs[0].Digest
			currentRef = appendedRefs[0].Name
			appendedRefs = appendedRefs[1:] // Skip first as it's now the baseline
		}
	}

	// Get line number mapping for the appended references
	lineNumbers, err := getLineNumbersForAppendedRefs(ctx, filePath, baseRef, headRef, baseCount, headCount)
	if err != nil {
		return nil, fmt.Errorf("get line numbers: %w", err)
	}

	// Detect digest transitions
	var transitions []StateTransition
	modulePath := filepath.Dir(filePath)

	for i, ref := range appendedRefs {
		if ref.Digest != currentDigest {
			// Digest changed! Record transition
			lineNumber := 0
			if i < len(lineNumbers) {
				lineNumber = lineNumbers[i]
			}

			transition := StateTransition{
				ModulePath: modulePath,
				FilePath:   filePath,
				FromRef:    currentRef,
				ToRef:      ref.Name,
				FromDigest: currentDigest,
				ToDigest:   ref.Digest,
				LineNumber: lineNumber,
			}
			transitions = append(transitions, transition)

			// Update current for next iteration
			currentDigest = ref.Digest
		}
		currentRef = ref.Name
	}

	return transitions, nil
}

// readFileAtRef reads a file's content at a specific git ref using git show.
func readFileAtRef(ctx context.Context, filePath string, ref string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", "show", fmt.Sprintf("%s:%s", ref, filePath))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s:%s: %w", ref, filePath, err)
	}
	return output, nil
}

// getLineNumbersForAppendedRefs calculates the line numbers in the diff where each
// appended reference's digest appears.
func getLineNumbersForAppendedRefs(
	ctx context.Context,
	filePath string,
	baseRef string,
	headRef string,
	baseCount int,
	headCount int,
) ([]int, error) {
	// Get the unified diff
	cmd := exec.CommandContext(ctx, "git", "diff", "-U0", baseRef, headRef, "--", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	return parseLineNumbersFromDiff(string(output), headCount-baseCount)
}

// parseLineNumbersFromDiff parses a git diff output and extracts line numbers
// where "digest" fields appear in added lines.
func parseLineNumbersFromDiff(diffOutput string, expectedCount int) ([]int, error) {
	lineNumbers := make([]int, expectedCount)
	scanner := bufio.NewScanner(strings.NewReader(diffOutput))

	var (
		currentLine    int
		refIndex       = 0
		inAddedSection = false
	)

	for scanner.Scan() {
		line := scanner.Text()

		// Parse hunk headers like: @@ -275,0 +276,12 @@
		if strings.HasPrefix(line, "@@") {
			parts := strings.Split(line, " ")
			if len(parts) >= 3 {
				// Extract new file line number from +276,12
				newRange := strings.TrimPrefix(parts[2], "+")
				newRange = strings.Split(newRange, ",")[0]
				fmt.Sscanf(newRange, "%d", &currentLine)
				inAddedSection = true
				continue
			}
		}

		if !inAddedSection {
			continue
		}

		// Look for added lines (starting with +)
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			// Check if this line contains "digest"
			if strings.Contains(line, `"digest"`) {
				if refIndex < len(lineNumbers) {
					lineNumbers[refIndex] = currentLine
					refIndex++
				}
			}
			currentLine++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan diff: %w", err)
	}

	return lineNumbers, nil
}
