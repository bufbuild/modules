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
	"fmt"
	"os"
	"strconv"
	"strings"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/pkg/slogapp"
	"github.com/bufbuild/modules/private/bufpkg/bufstate"
	"github.com/spf13/pflag"
)

const rootCmdName = "commentprcasdiff"

func main() {
	appcmd.Main(context.Background(), newCommand(rootCmdName))
}

func newCommand(name string) *appcmd.Command {
	builder := appext.NewBuilder(
		name,
		appext.BuilderWithLoggerProvider(slogapp.LoggerProvider),
	)
	flags := newFlags()
	return &appcmd.Command{
		Use:       name,
		Short:     "Post CAS diff comments on PRs when module digests change.",
		BindFlags: flags.bind,
		Run: builder.NewRunFunc(
			func(ctx context.Context, _ appext.Container) error {
				return run(ctx, flags)
			},
		),
	}
}

type flags struct {
	dryRun bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) bind(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.dryRun, "dry-run", false, "print comments to stdout instead of posting to GitHub")
}

func run(ctx context.Context, flags *flags) error {
	baseRef := os.Getenv("BASE_REF")
	headRef := os.Getenv("HEAD_REF")
	prNumberString := os.Getenv("PR_NUMBER")
	var prNumber int

	if baseRef == "" {
		return errors.New("BASE_REF environment variable is required")
	}
	if headRef == "" {
		return errors.New("HEAD_REF environment variable is required")
	}
	if !flags.dryRun {
		if os.Getenv("GITHUB_TOKEN") == "" {
			return errors.New("GITHUB_TOKEN environment variable is required when not a dry-run")
		}
		if prNumberString == "" {
			return errors.New("PR_NUMBER environment variable is required when not a dry-run")
		}
		var err error
		prNumber, err = strconv.Atoi(prNumberString)
		if err != nil {
			return fmt.Errorf("parse PR_NUMBER: %w", err)
		}
	}

	fmt.Fprintf(
		os.Stdout,
		"Processing PR #%s (base: %s, head: %s)\n",
		cmp.Or(prNumberString, "dry-run"),
		baseRef,
		headRef,
	)

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

	stateRW, err := bufstate.NewReadWriter()
	if err != nil {
		return fmt.Errorf("new state read writer: %w", err)
	}

	var allTransitions []stateTransition
	for _, moduleStatePath := range moduleStatePathsSorted {
		fmt.Fprintf(os.Stdout, "Analyzing %s...\n", moduleStatePath)

		transitions, err := getStateFileTransitions(ctx, stateRW, moduleStatePath, baseRef, headRef)
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

	overallTransitions, err := getGlobalOverallTransitions(ctx, stateRW, baseRef, headRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get overall transitions from global state: %v\n", err)
	} else if len(overallTransitions) > 0 {
		fmt.Fprintf(os.Stdout, "Found %d overall transition(s) in global state:\n", len(overallTransitions))
		for _, t := range overallTransitions {
			fmt.Fprintf(os.Stdout, "  %s: %s -> %s\n", strings.TrimPrefix(t.modulePath, "modules/sync/"), t.fromRef, t.toRef)
		}
		allTransitions = append(allTransitions, overallTransitions...)
	}

	if len(allTransitions) == 0 {
		fmt.Fprintf(os.Stdout, "No digest transitions found\n")
		return nil
	}

	fmt.Fprintf(os.Stdout, "\nRunning casdiff for %d transition(s)...\n", len(allTransitions))
	results := runCASDiffs(ctx, allTransitions)

	var (
		casResults   []casDiffResult
		errsToReturn []error
	)

	for _, result := range results {
		if result.err != nil {
			errsToReturn = append(
				errsToReturn,
				fmt.Errorf(
					"casdiff failed for %s %s->%s: %w",
					result.transition.modulePath,
					result.transition.fromRef,
					result.transition.toRef,
					result.err,
				),
			)
		} else {
			casResults = append(casResults, result)
		}
	}

	if len(casResults) > 0 {
		if flags.dryRun {
			fmt.Fprintf(os.Stdout, "\n[dry-run] %d comment(s) would be posted:\n", len(casResults))
			for _, result := range casResults {
				fmt.Fprintf(os.Stdout, "\n--- %s (line %d: %s -> %s) ---\n%s\n",
					result.transition.filePath,
					result.transition.lineNumber,
					result.transition.fromRef,
					result.transition.toRef,
					result.output,
				)
			}
		} else {
			fmt.Fprintf(os.Stdout, "\nPosting %d comment(s) to PR...\n", len(casResults))
			comments := make([]prReviewComment, len(casResults))
			for i, result := range casResults {
				comments[i] = prReviewComment{
					filePath:   result.transition.filePath,
					lineNumber: result.transition.lineNumber,
					body:       result.output,
				}
			}
			if err := postReviewComments(ctx, prNumber, headRef, comments...); err != nil {
				errsToReturn = append(errsToReturn, fmt.Errorf("post review comments: %w", err))
			}
		}
	}

	if len(errsToReturn) > 0 {
		return errors.Join(errsToReturn...)
	}
	fmt.Fprintf(os.Stdout, "\nDone!\n")
	return nil
}
