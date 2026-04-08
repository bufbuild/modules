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
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bufbuild/modules/private/bufpkg/bufstate"
	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
)

// stateTransition represents a digest change in a module's state.json file.
type stateTransition struct {
	modulePath         string // e.g., "modules/sync/bufbuild/protovalidate"
	filePath           string // The module where the transition happened, can be an individual module like "modules/sync/bufbuild/protovalidate/state.json" or the global state file "modules/sync/state.json"
	fromRef            string // Old git reference (e.g., "v1.1.0")
	toRef              string // New git reference (e.g., "v1.2.0")
	fromDigest         string // Old digest
	toDigest           string // New digest
	lineNumber         int    // Line in diff where the new reference or digest appears.
	isGlobalTransition bool   // True for the transition on the state.json global to all modules.
}

// getStateFileTransitions reads state.json from base and head branches, compares the JSON arrays to
// find appended references, and detects digest transitions.
func getStateFileTransitions(
	ctx context.Context,
	stateRW *bufstate.ReadWriter,
	filePath string,
	baseRef string,
	headRef string,
) ([]stateTransition, error) {
	// Read state.json from both branches
	baseContent, err := readFileAtRef(ctx, filePath, baseRef)
	if err != nil {
		return nil, fmt.Errorf("read base state: %w", err)
	}
	headContent, err := readFileAtRef(ctx, filePath, headRef)
	if err != nil {
		return nil, fmt.Errorf("read head state: %w", err)
	}

	baseState, err := stateRW.ReadModStateFile(io.NopCloser(bytes.NewReader(baseContent)))
	if err != nil {
		return nil, fmt.Errorf("parse base state: %w", err)
	}
	headState, err := stateRW.ReadModStateFile(io.NopCloser(bytes.NewReader(headContent)))
	if err != nil {
		return nil, fmt.Errorf("parse head state: %w", err)
	}

	// Identify appended references.
	//
	// This only works for diffs that append references in the state file, not for diffs that
	// update/remove existing refs in base.
	baseRefs := baseState.GetReferences()
	headRefs := headState.GetReferences()
	current, appendedRefs := resolveAppendedRefs(baseRefs, headRefs)
	if current == nil {
		return nil, nil
	}

	// Get line number mapping for the appended references
	lineNumbers, err := getLineNumbersForAppendedRefs(ctx, filePath, baseRef, headRef, len(baseRefs), len(headRefs))
	if err != nil {
		return nil, fmt.Errorf("get line numbers: %w", err)
	}

	// Detect digest transitions
	var (
		modulePath    = filepath.Dir(filePath)
		currentRef    = current.GetName()
		currentDigest = current.GetDigest()
		transitions   []stateTransition
	)
	for i, appendedRef := range appendedRefs {
		if appendedRef.GetDigest() != currentDigest {
			// Digest changed! Record transition
			var lineNumber int
			if i < len(lineNumbers) {
				lineNumber = lineNumbers[i]
			}
			transitions = append(transitions, stateTransition{
				modulePath:         modulePath,
				filePath:           filePath,
				fromRef:            currentRef,
				toRef:              appendedRef.GetName(),
				fromDigest:         currentDigest,
				toDigest:           appendedRef.GetDigest(),
				lineNumber:         lineNumber,
				isGlobalTransition: false,
			})
			currentDigest = appendedRef.GetDigest()
		}
		currentRef = appendedRef.GetName()
	}

	return transitions, nil
}

// resolveAppendedRefs identifies the refs from headRefs that are new, considering new by
// index-behavior, not its content. Meaning if base has 3 refs and head has 5, we assume the first 3
// in head are the same 3 in base, and the latest 2 are "appended". That is the use case for the
// fetch-modules PR, and that is for now the only supported behavior.
//
// Returns (nil, nil) when the head refs is empty. The current ref is the latest in the base refs.
// If base is empty, the first ref in head is returned as the current, and all the rest are returned
// as appended.
func resolveAppendedRefs(
	baseRefs []*statev1alpha1.ModuleReference,
	headRefs []*statev1alpha1.ModuleReference,
) (*statev1alpha1.ModuleReference, []*statev1alpha1.ModuleReference) {
	if len(baseRefs) == 0 && len(headRefs) == 0 {
		// both empty
		//   - current: none
		//   - appended: none
		return nil, nil
	}
	if len(baseRefs) == 0 {
		// base empty
		//   - current: first in head
		//   - appended: the rest from head (if any)
		return headRefs[0], headRefs[1:]
	}
	if len(headRefs) <= len(baseRefs) {
		// head has the same amount or less refs than base
		//   - current: last in base
		//   - appended: none
		return baseRefs[len(baseRefs)-1], nil
	}
	// head has more refs than base (expected scenario most of the times)
	//   - current: last in base
	//   - appended: the index-base appended in head
	return baseRefs[len(baseRefs)-1], headRefs[len(baseRefs):]
}

// getGlobalOverallTransitions reads modules/sync/state.json from both base and head, compares the
// two, and returns one stateTransition per module whose latest_reference changed. Modules that were
// added or removed between base and head are ignored.
func getGlobalOverallTransitions(
	ctx context.Context,
	stateRW *bufstate.ReadWriter,
	baseRef string,
	headRef string,
) ([]stateTransition, error) {
	const globalStatePath = "modules/sync/state.json"

	baseContent, err := readFileAtRef(ctx, globalStatePath, baseRef)
	if err != nil {
		return nil, fmt.Errorf("read base global state: %w", err)
	}
	headContent, err := readFileAtRef(ctx, globalStatePath, headRef)
	if err != nil {
		return nil, fmt.Errorf("read head global state: %w", err)
	}

	baseGlobalState, err := stateRW.ReadGlobalState(io.NopCloser(bytes.NewReader(baseContent)))
	if err != nil {
		return nil, fmt.Errorf("parse base global state: %w", err)
	}
	headGlobalState, err := stateRW.ReadGlobalState(io.NopCloser(bytes.NewReader(headContent)))
	if err != nil {
		return nil, fmt.Errorf("parse head global state: %w", err)
	}

	baseLatestRefs := make(map[string]string, len(baseGlobalState.GetModules()))
	for _, mod := range baseGlobalState.GetModules() {
		baseLatestRefs[mod.GetModuleName()] = mod.GetLatestReference()
	}

	var transitions []stateTransition
	for _, mod := range headGlobalState.GetModules() {
		moduleName := mod.GetModuleName()
		toRef := mod.GetLatestReference()
		fromRef, existsInBase := baseLatestRefs[moduleName]
		if !existsInBase || fromRef == toRef {
			continue // it is a new module, or the reference did not change.
		}
		lineNumber, err := findLatestReferenceLineInGlobalState(headContent, moduleName)
		if err != nil {
			return nil, fmt.Errorf("find line number for %q: %w", moduleName, err)
		}
		transitions = append(transitions, stateTransition{
			modulePath:         "modules/sync/" + moduleName,
			filePath:           globalStatePath,
			fromRef:            fromRef,
			toRef:              toRef,
			lineNumber:         lineNumber,
			isGlobalTransition: true,
		})
	}
	return transitions, nil
}

// findLatestReferenceLineInGlobalState scans the raw JSON of modules/sync/state.json and returns
// the 1-based line number of the "latest_reference" field for the given module.
func findLatestReferenceLineInGlobalState(content []byte, moduleName string) (int, error) {
	quotedName := `"` + moduleName + `"`
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var (
		lineNum     int
		foundModule bool
	)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if !foundModule {
			if strings.Contains(line, `"module_name"`) && strings.Contains(line, quotedName) {
				foundModule = true
			}
		} else if strings.Contains(line, `"latest_reference"`) {
			return lineNum, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scan global state: %w", err)
	}
	return 0, fmt.Errorf("latest_reference for module %q not found in global state", moduleName)
}

// readFileAtRef reads a file's content at a specific git ref using git show.
func readFileAtRef(ctx context.Context, filePath string, ref string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", "show", fmt.Sprintf("%s:%s", ref, filePath)) //nolint:gosec
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s:%s: %w", ref, filePath, err)
	}
	return output, nil
}

// getLineNumbersForAppendedRefs calculates the line numbers in the diff where each appended
// reference's digest appears.
func getLineNumbersForAppendedRefs(
	ctx context.Context,
	filePath string,
	baseRef string,
	headRef string,
	baseCount int,
	headCount int,
) ([]int, error) {
	// Get the unified diff, with zero lines of context up or down.
	cmd := exec.CommandContext(ctx, "git", "diff", "-U0", baseRef, headRef, "--", filePath) //nolint:gosec
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return parseLineNumbersFromDiff(string(output), headCount-baseCount)
}

// parseLineNumbersFromDiff parses a git diff output and extracts line numbers where "digest" fields
// appear in added lines.
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
				if _, err := fmt.Sscanf(newRange, "%d", &currentLine); err != nil {
					return nil, fmt.Errorf("parse hunk header line number %q: %w", newRange, err)
				}
				inAddedSection = true
				continue
			}
		}
		if !inAddedSection {
			continue
		}
		// Look for added lines (starting with +) with the string `"digest":`
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			if strings.Contains(line, `"digest":`) {
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
