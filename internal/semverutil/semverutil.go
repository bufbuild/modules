// Copyright 2021-2023 Buf Technologies, Inc.
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

package semverutil

import (
	"fmt"
	"sort"

	"golang.org/x/mod/semver"
)

// SemverTagNames gets the valid semver tag names from the list.
func SemverTagNames(tagNames []string) []string {
	semverTagNames := make([]string, 0, len(tagNames))
	for _, tagName := range tagNames {
		if semver.IsValid(tagName) {
			semverTagNames = append(semverTagNames, tagName)
		}
	}
	return semverTagNames
}

// ValidateSemverTagNames validates that all tag names in the list are valid semver tag names.
func ValidateSemverTagNames(tagNames []string) error {
	for _, tagName := range tagNames {
		if !semver.IsValid(tagName) {
			return fmt.Errorf("%q is not a valid SemVer tag name", tagName)
		}
	}
	return nil
}

// StableSemverTagNames gets the stable semver tag names from the list.
func StableSemverTagNames(semverTagNames []string) []string {
	stableTagNames := make([]string, 0, len(semverTagNames))
	for _, semverTagName := range semverTagNames {
		if semver.Prerelease(semverTagName) == "" && semver.Build(semverTagName) == "" {
			stableTagNames = append(stableTagNames, semverTagName)
		}
	}
	return stableTagNames
}

// SemverTagNamesAtLeast gets the semver tag names that are at least the minimum tag name.
func SemverTagNamesAtLeast(semverTagNames []string, minimumSemverTagName string, inclusive bool) []string {
	atLeastTagNames := make([]string, 0, len(semverTagNames))
	for _, semverTagName := range semverTagNames {
		compare := semver.Compare(semverTagName, minimumSemverTagName)
		if compare > 0 || (compare == 0 && inclusive) {
			atLeastTagNames = append(atLeastTagNames, semverTagName)
		}
	}
	return atLeastTagNames
}

// SemverTagNamesExcept gets the semver tag names minus the skip tag names.
func SemverTagNamesExcept(semverTagNames []string, skipTagNames map[string]struct{}) []string {
	exceptTagNames := make([]string, 0, len(semverTagNames))
	for _, semverTagName := range semverTagNames {
		if _, ok := skipTagNames[semverTagName]; !ok {
			exceptTagNames = append(exceptTagNames, semverTagName)
		}
	}
	return exceptTagNames
}

// SortSemverTagNames sorts the semver tag names.
func SortSemverTagNames(semverTagNames []string) {
	sort.Slice(
		semverTagNames,
		func(i int, j int) bool {
			return semver.Compare(
				semverTagNames[i],
				semverTagNames[j],
			) < 0
		},
	)
}
