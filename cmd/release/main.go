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
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bufbuild/modules/internal/githubutil"
	"github.com/bufbuild/modules/internal/modules"
	"github.com/bufbuild/modules/private/bufpkg/bufstate"
	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
	"github.com/google/go-github/v64/github"
	"go.uber.org/multierr"
)

type command struct {
	dryRun bool
}

type releaseModuleState struct {
	status     modules.Status
	references []*statev1alpha1.ModuleReference
}

func main() {
	dryRun := flag.Bool("dry-run", false, "perform a dry-run (no GitHub modifications)")
	flag.Parse()

	if len(flag.Args()) != 1 {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "usage: release <directory>")
		flag.PrintDefaults()
		os.Exit(2)
	}
	cmd := &command{
		dryRun: *dryRun,
	}
	if err := cmd.run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "run release : %v\n", err)
		os.Exit(1)
	}
}

func (c *command) run() (retErr error) {
	tmpDir, err := os.MkdirTemp("", "modules-release")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	_, _ = fmt.Fprintf(os.Stdout, "created tmp dir: %s\n", tmpDir)
	defer func() {
		if c.dryRun {
			return
		}
		if err := os.RemoveAll(tmpDir); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("remove %q: %w", tmpDir, err))
		}
	}()
	ctx := context.Background()
	githubClient := githubutil.NewClient(ctx)
	prevRelease, err := githubClient.GetLatestRelease(ctx, githubutil.GithubOwnerBufbuild, githubutil.GithubRepoModules)
	if err != nil && !errors.Is(err, githubutil.ErrNotFound) {
		return fmt.Errorf("retrieve latest release: %w", err)
	}
	var prevReleaseState *statev1alpha1.GlobalState
	if prevRelease != nil {
		prevReleaseState, err = githubClient.DownloadReleaseState(ctx, prevRelease)
		if err != nil {
			return err
		}
	}
	stateRW, err := bufstate.NewReadWriter()
	if err != nil {
		return fmt.Errorf("new state read writer: %w", err)
	}
	globalStateFilePath := filepath.Join(bufstate.SyncRoot, bufstate.GlobalStateFileName)
	var currentReleaseState *statev1alpha1.GlobalState
	if _, err := os.Stat(globalStateFilePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat file: %w", err)
		}
		currentReleaseState = &statev1alpha1.GlobalState{}
	} else {
		globalStateFile, err := os.Open(globalStateFilePath)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		currentReleaseState, err = stateRW.ReadGlobalState(globalStateFile)
		if err != nil {
			return fmt.Errorf("read local state file: %w", err)
		}
	}
	prevMap := mapGlobalStateReferences(prevReleaseState)
	currentMap := mapGlobalStateReferences(currentReleaseState)
	modulesStates, err := calculateModulesStates(stateRW, bufstate.SyncRoot, prevMap, currentMap)
	if err != nil {
		return fmt.Errorf("produce new module list: %w", err)
	}
	if !shouldRelease(modulesStates) {
		errMsg := "no changes to modules - not creating initial release"
		if tagName := prevRelease.GetTagName(); tagName != "" {
			errMsg = fmt.Sprintf("no changes to modules since %v", tagName)
		}
		_, _ = fmt.Fprintln(os.Stdout, errMsg)
		return nil
	}
	now := time.Now().Truncate(time.Second)
	releaseName, err := calculateNextRelease(now, prevRelease)
	if err != nil {
		return fmt.Errorf("determine next release name: %w", err)
	}
	if c.dryRun {
		releaseBody, err := createReleaseBody(releaseName, modulesStates)
		if err != nil {
			return fmt.Errorf("create release body: %w", err)
		}
		_, _ = fmt.Fprintln(os.Stdout, releaseBody)
		_, _ = fmt.Fprintln(os.Stdout, "skipping GitHub release creation in dry-run mode")
		_, _ = fmt.Fprintf(os.Stdout, "release assets created in %q\n", tmpDir)
		return nil
	}
	if err := createRelease(ctx, githubClient, releaseName, modulesStates); err != nil {
		return fmt.Errorf("create GitHub release: %w", err)
	}
	return nil
}

func mapGlobalStateReferences(globalState *statev1alpha1.GlobalState) map[string]string {
	if globalState == nil || len(globalState.GetModules()) == 0 {
		return nil
	}
	stateMap := make(map[string]string, len(globalState.GetModules()))
	for _, module := range globalState.GetModules() {
		stateMap[module.GetModuleName()] = module.GetLatestReference()
	}
	return stateMap
}

// calculateModulesStates accepts the repository's checked out module manifest file `current`, as
// well as the `prev` from the latest published release. It will build a list of all modules
// `updatedModules` that have not yet been released and the last version of that module, if present,
// that was released. Using the `updatedModules` list, it will load the individual module's manifest
// file from the module`s directory, and determine which versions of the module have not been
// released by comparing it against the last released version of the module. The resultant list
// tells us which modules and which of their references have not been released, and the state of
// each of the modules, whether they're new, updated, or unchanged.
func calculateModulesStates(
	stateRW *bufstate.ReadWriter,
	dir string,
	prev map[string]string,
	current map[string]string,
) (map[string]releaseModuleState, error) {
	moduleReferences := make(map[string]releaseModuleState)
	var updatedModules []modules.Module
	// If the latest manifest doesn't exist, everything is new
	if prev == nil {
		for moduleName := range current {
			updatedModules = append(updatedModules, modules.Module{Name: moduleName})
		}
	} else {
		// otherwise, determine individually new modules
		seenModules := make(map[string]struct{}, len(current))
		// compare current vs previous to get [new, updated, unchanged] modules
		for moduleName, currentReference := range current {
			seenModules[moduleName] = struct{}{}
			prevReleaseReference, ok := prev[moduleName]
			if !ok {
				// module not present in the previous state, it's new
				updatedModules = append(updatedModules, modules.Module{
					Name: moduleName,
					// no LastReleasedReference means New
				})
				continue
			}
			if prevReleaseReference == currentReference {
				// module reference did not change, it's unchanged
				moduleReferences[moduleName] = releaseModuleState{status: modules.Unchanged}
				continue
			}
			// module reference changed vs previous state, it's changed
			updatedModules = append(updatedModules, modules.Module{
				Name:                  moduleName,
				LastReleasedReference: prevReleaseReference,
			})
		}
		// compare previous vs current to get removed modules
		for moduleName := range prev {
			if _, seen := seenModules[moduleName]; !seen {
				// module present in previous but not in current, it's removed
				moduleReferences[moduleName] = releaseModuleState{status: modules.Removed}
				seenModules[moduleName] = struct{}{}
			}
		}
	}

	for _, updatedModule := range updatedModules {
		modFilePath := filepath.Join(dir, updatedModule.Name, bufstate.ModStateFileName)
		var moduleManifest *statev1alpha1.ModuleState
		if _, err := os.Stat(modFilePath); err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("stat file: %w", err)
			}
			moduleManifest = &statev1alpha1.ModuleState{}
		} else {
			modStateFile, err := os.Open(modFilePath)
			if err != nil {
				return nil, fmt.Errorf("open file: %w", err)
			}
			moduleManifest, err = stateRW.ReadModStateFile(modStateFile)
			if err != nil {
				return nil, fmt.Errorf("retrieve module state: %w", err)
			}
		}
		if updatedModule.LastReleasedReference == "" {
			moduleReferences[updatedModule.Name] = releaseModuleState{
				status:     modules.New,
				references: moduleManifest.GetReferences(),
			}
			continue
		}
		for index, reference := range moduleManifest.GetReferences() {
			if reference.GetName() == updatedModule.LastReleasedReference {
				if len(moduleManifest.GetReferences()) == index-1 {
					return nil, fmt.Errorf("module indicated as having updates, but previous release %s is the latest", updatedModule.LastReleasedReference)
				}
				moduleReferences[updatedModule.Name] = releaseModuleState{
					status:     modules.Updated,
					references: moduleManifest.GetReferences()[index+1:],
				}
				break
			}
		}
		if _, ok := moduleReferences[updatedModule.Name]; !ok {
			// if no match was found, it means the previous latest reference was gone, so take all refs
			// as updated
			moduleReferences[updatedModule.Name] = releaseModuleState{
				status:     modules.Updated,
				references: moduleManifest.GetReferences(),
			}
		}
	}
	return moduleReferences, nil
}

func calculateNextRelease(now time.Time, prevRelease *github.RepositoryRelease) (string, error) {
	var prevReleaseName string
	if prevRelease != nil {
		prevReleaseName = prevRelease.GetTagName()
	}
	currentDate := now.UTC().Format("20060102")
	if prevRelease == nil || !strings.HasPrefix(prevReleaseName, currentDate+".") {
		return currentDate + ".1", nil
	}
	_, revision, ok := strings.Cut(prevReleaseName, ".")
	if !ok {
		return "", fmt.Errorf("malformed latest release tag name: %v", prevRelease.GetTagName())
	}
	currentRevision, err := strconv.Atoi(revision)
	if err != nil {
		return "", err
	}
	return currentDate + "." + strconv.Itoa(currentRevision+1), nil
}

// shouldRelease checks if any module status is different than "unchanged", so we do releases for
// new, updated, or removed modules.
func shouldRelease(modulesStates map[string]releaseModuleState) bool {
	for _, state := range modulesStates {
		if state.status != modules.Unchanged {
			return true
		}
	}
	return false
}

func createRelease(
	ctx context.Context,
	client *githubutil.Client,
	releaseName string,
	modules map[string]releaseModuleState,
) error {
	releaseBody, err := createReleaseBody(releaseName, modules)
	if err != nil {
		return fmt.Errorf("create release body: %w", err)
	}
	repositoryRelease, err := client.CreateRelease(ctx,
		githubutil.GithubOwnerBufbuild,
		githubutil.GithubRepoModules,
		&github.RepositoryRelease{
			TagName: github.String(releaseName),
			Name:    github.String(releaseName),
			Body:    github.String(releaseBody),
			Draft:   github.Bool(true), // Start release as a draft until all assets are uploaded
		})
	if err != nil {
		return err
	}
	// Upload the single state.json artifact
	if err := client.UploadReleaseAsset(ctx,
		githubutil.GithubOwnerBufbuild,
		githubutil.GithubRepoModules,
		repositoryRelease.GetID(),
		filepath.Join(bufstate.SyncRoot, bufstate.GlobalStateFileName),
	); err != nil {
		return err
	}
	repositoryRelease.Draft = github.Bool(false)
	if _, err := client.EditRelease(ctx,
		githubutil.GithubOwnerBufbuild,
		githubutil.GithubRepoModules,
		repositoryRelease.GetID(),
		repositoryRelease,
	); err != nil {
		return err
	}
	return nil
}

func createReleaseBody(name string, moduleStates map[string]releaseModuleState) (string, error) {
	var mainStringBuilder strings.Builder
	if _, err := mainStringBuilder.WriteString(fmt.Sprintf("# Buf Modules Release %s\n\n", name)); err != nil {
		return "", err
	}

	sortedModNames := make([]string, 0, len(moduleStates))
	for modName := range moduleStates {
		sortedModNames = append(sortedModNames, modName)
	}
	sort.Slice(sortedModNames, func(i, j int) bool {
		return sortedModNames[i] < sortedModNames[j]
	})
	var newStringBuilder, updatedStringBuilder, unchangedStringBuilder, removedStringBuilder strings.Builder
	for _, modName := range sortedModNames {
		modState := moduleStates[modName]
		switch modState.status {
		case modules.New:
			if err := writeUpdatedReferencesTable(&newStringBuilder, modName, modState.references); err != nil {
				return "", fmt.Errorf("write new modules table: %w", err)
			}
		case modules.Updated:
			if err := writeUpdatedReferencesTable(&updatedStringBuilder, modName, modState.references); err != nil {
				return "", fmt.Errorf("write updated modules table: %w", err)
			}
		case modules.Unchanged:
			if _, err := unchangedStringBuilder.WriteString(fmt.Sprintf("- %s\n", modName)); err != nil {
				return "", err
			}
		case modules.Removed:
			if _, err := removedStringBuilder.WriteString(fmt.Sprintf("- %s\n", modName)); err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("module %s has not set a release state", modName)
		}
	}

	if newStr := newStringBuilder.String(); newStr != "" {
		newModuleHeader := "## New Modules\n"
		if _, err := mainStringBuilder.WriteString(fmt.Sprintf("%s%s\n", newModuleHeader, newStr)); err != nil {
			return "", err
		}
	}

	if updated := updatedStringBuilder.String(); updated != "" {
		updatedModuleHeader := "## Updated Modules\n"
		if _, err := mainStringBuilder.WriteString(fmt.Sprintf("%s%s\n", updatedModuleHeader, updated)); err != nil {
			return "", err
		}
	}

	if unchanged := unchangedStringBuilder.String(); unchanged != "" {
		unchangedModuleHeader := "## Unchanged Modules\n\n<details><summary>Expand</summary>\n"
		if _, err := mainStringBuilder.WriteString(fmt.Sprintf("%s\n%s\n</details>\n", unchangedModuleHeader, unchanged)); err != nil {
			return "", err
		}
	}

	if removed := removedStringBuilder.String(); removed != "" {
		removedModuleHeader := "## Removed Modules\n\n<details><summary>Expand</summary>\n"
		if _, err := mainStringBuilder.WriteString(fmt.Sprintf("\n%s\n%s\n</details>\n", removedModuleHeader, removed)); err != nil {
			return "", err
		}
	}

	return mainStringBuilder.String(), nil
}

func writeUpdatedReferencesTable(
	stringBuilder *strings.Builder,
	moduleName string,
	references []*statev1alpha1.ModuleReference,
) error {
	refCount := len(references)
	if _, err := fmt.Fprintf(stringBuilder,
		"\n<details><summary>%s: %d update(s)</summary>\n\n| Reference | Manifest Digest |\n|---|---|\n",
		moduleName, refCount,
	); err != nil {
		return err
	}
	const (
		topRows    = 2
		bottomRows = 2
		// maxRows is the maximum amount of reference rows in a table
		maxRows = topRows + bottomRows + 1 // +1 middle row saying something was skipped
	)
	var (
		refsToWrite []*statev1alpha1.ModuleReference
		fitsInTable = refCount <= maxRows
	)
	if fitsInTable {
		refsToWrite = references // write them all
	} else {
		refsToWrite = references[0:topRows]
		refsToWrite = append(refsToWrite, references[refCount-bottomRows:]...)
	}
	// write table
	for i, ref := range refsToWrite {
		if !fitsInTable && i == topRows {
			skippedRows := refCount - topRows - bottomRows
			if _, err := fmt.Fprintf(stringBuilder,
				"| ... %d references skipped ... | ... %d references skipped ... |\n",
				skippedRows, skippedRows,
			); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(stringBuilder, "| `%s` | `%s` |\n", ref.GetName(), ref.GetDigest()); err != nil {
			return err
		}
	}
	if _, err := stringBuilder.WriteString("\n</details>\n"); err != nil {
		return err
	}
	return nil
}
