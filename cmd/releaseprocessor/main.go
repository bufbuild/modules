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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/modules/internal/githubutil"
	"go.uber.org/multierr"
	"golang.org/x/mod/semver"
)

const (
	ownerFlagName     = "owner"
	repoFlagName      = "repo"
	referenceFlagName = "reference"
	inclusiveFlagName = "inclusive"
)

var (
	skipTags = map[string]map[string]struct{}{ //nolint:gochecknoglobals
		"protocolbuffers/protobuf": {
			"v3.4.1": {}, // v3.4.1 did not have protoc attached to it
		},
		"bufbuild/protovalidate-testing": {
			"v0.8.0": {}, // v0.8.0 is (intentionally) broken
		},
	}
	// only allow unstable semver releases if they're release candidates in the shape of v1.2.3-rc.4
	allowedPreReleaseRx = regexp.MustCompile(`^-rc\.\d+$`)
)

type command struct {
	owner     string
	repo      string
	reference string
	inclusive bool
}

func newCmd(
	owner string,
	repo string,
	reference string,
	inclusive bool,
) (*command, error) {
	var err error
	if len(owner) == 0 {
		err = multierr.Append(err, fmt.Errorf("%s is required", ownerFlagName))
	}
	if len(repo) == 0 {
		err = multierr.Append(err, fmt.Errorf("%s is required", repoFlagName))
	}
	if len(reference) == 0 {
		err = multierr.Append(err, fmt.Errorf("%s is required", referenceFlagName))
	}
	if err != nil {
		return nil, err
	}
	return &command{
		owner:     owner,
		repo:      repo,
		reference: reference,
		inclusive: inclusive,
	}, nil
}

func main() {
	var (
		owner     = flag.String(ownerFlagName, "", "Managed module owner name.")
		repo      = flag.String(repoFlagName, "", "Managed module repository name.")
		reference = flag.String(referenceFlagName, "", "Managed module reference to fetch new release tags from.")
		inclusive = flag.Bool(inclusiveFlagName, false, "Include passed reference in the output revision list.")
	)
	flag.Parse()
	cmd, err := newCmd(*owner, *repo, *reference, *inclusive)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cannot run release processor: %v\n\nusage: releaseprocessor [flags]\n\n", err)
		flag.PrintDefaults()
		os.Exit(2)
	}
	if err := cmd.run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "releaseprocessor failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// run executes a releaseprocessor job, which fetches all tagged releases from the Github's API,
// filters only stable semver tags, filters out skipped semver tags defined in a global variable,
// sorts them in ascending order (semver-sort, not timestamp sort), and prints them out.
func (c *command) run() error {
	ctx := context.Background()
	githubClient := githubutil.NewClient(ctx)
	// N.B. these are _release_ tags, which differs from all tags.
	releaseTagNames, err := githubClient.AllReleaseTagNames(ctx, c.owner, c.repo)
	if err != nil {
		return fmt.Errorf("fetch all release tag names: %w", err)
	}
	filteredTags := c.filterReleaseTags(releaseTagNames)
	// write the release tags to stdout, separated by line breaks so that
	// assignment to a bash variable can interpret it as a list
	if _, err := os.Stdout.WriteString(strings.Join(filteredTags, "\n")); err != nil {
		return fmt.Errorf("write release tags to stdout: %w", err)
	}
	return nil
}

// filterReleaseTags takes a list of released git tags, filters them to only include valid semver
// tags, uses [c.reference] and [c.inclusive], and returns them semver-sorted.
func (c *command) filterReleaseTags(releaseTags []string) []string {
	tags := xslices.Filter(
		releaseTags,
		func(tag string) bool {
			if skippableTags, ok := skipTags[filepath.Join(c.owner, c.repo)]; ok {
				if _, shouldSkip := skippableTags[tag]; shouldSkip {
					return false
				}
			}
			if !semver.IsValid(tag) {
				return false
			}
			if semver.Build(tag) != "" {
				return false
			}
			if preRelease := semver.Prerelease(tag); preRelease != "" {
				if !allowedPreReleaseRx.MatchString(preRelease) {
					return false
				}
			}
			switch compare := semver.Compare(tag, c.reference); {
			case compare < 0:
				return false
			case compare == 0:
				return c.inclusive
			default: // compare > 0
				return true
			}
		},
	)
	semver.Sort(tags)
	return tags
}
