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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bufbuild/modules/internal/githubutil"
	"github.com/bufbuild/modules/internal/modules"
	"github.com/bufbuild/modules/private/bufpkg/bufstate"
	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
	"github.com/google/go-github/v48/github"
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
	_, _ = fmt.Fprintf(os.Stdout, "created tmp dir: %s", tmpDir)
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
		prevReleaseState, err = githubClient.DownloadReleaseManifest(ctx, prevRelease)
		if err != nil {
			return err
		}
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
		currentReleaseState, err = bufstate.ReadGlobalState(globalStateFile)
		if err != nil {
			return fmt.Errorf("read local state file: %w", err)
		}
	}
	prevMap := mapGlobalStateReferences(prevReleaseState)
	currentMap := mapGlobalStateReferences(currentReleaseState)
	newModuleReleases, err := calculateNewReleaseModules(bufstate.SyncRoot, prevMap, currentMap)
	if err != nil {
		return fmt.Errorf("produce new module list: %w", err)
	}
	if len(newModuleReleases) == 0 {
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
		releaseBody, err := createReleaseBody(releaseName, newModuleReleases)
		if err != nil {
			return fmt.Errorf("create release body: %w", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "RELEASE.md"), []byte(releaseBody), 0644); err != nil { //nolint:gosec
			return err
		}
		_, _ = fmt.Fprintln(os.Stdout, "skipping GitHub release creation in dry-run mode")
		_, _ = fmt.Fprintf(os.Stdout, "release assets created in %q", tmpDir)
		return nil
	}
	if err := createRelease(ctx, githubClient, releaseName, newModuleReleases); err != nil {
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

// calculateNewReleaseModules accepts the repository's checked out module manifest file `current`, as well
// as the `prev` from the latest published release. It will build a list of all modules `updatedModules`
// that have not yet been released and the last version of that module, if present, that was released.
// Using the `updatedModules` list, it will load the individual module's manifest file from the module`s directory,
// and determine which versions of the module have not been released by comparing it against the last released version
// of the module. The resultant list tells us which modules and which of their references have not been released, and
// the state of each of the modules, whether they're new, updated, or unchanged.
func calculateNewReleaseModules(
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
		for moduleName, currentReference := range current {
			prevReleaseReference, ok := prev[moduleName]
			if !ok {
				// new
				updatedModules = append(updatedModules, modules.Module{Name: moduleName})
				continue
			}
			// unchanged, record it now as we don't use the source of this information in the next step
			if prevReleaseReference == currentReference {
				moduleReferences[moduleName] = releaseModuleState{status: modules.Unchanged}
				continue
			}
			// updated
			updatedModules = append(updatedModules, modules.Module{
				Name:                  moduleName,
				LastReleasedReference: prevReleaseReference,
			})
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
			moduleManifest, err = bufstate.ReadModStateFile(modStateFile)
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

	var newStringBuilder, updatedStringBuilder, unchangedStringBuilder strings.Builder
	for module, modState := range moduleStates {
		switch modState.status {
		case modules.New:
			if err := writeUpdatedReferencesTable(&newStringBuilder, module, modState.references); err != nil {
				return "", fmt.Errorf("write new modules table: %w", err)
			}
		case modules.Updated:
			if err := writeUpdatedReferencesTable(&updatedStringBuilder, module, modState.references); err != nil {
				return "", fmt.Errorf("write updated modules table: %w", err)
			}
		case modules.Unchanged:
			if _, err := unchangedStringBuilder.WriteString(fmt.Sprintf("- %s\n", module)); err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("module %s has not set a release state", module)
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
		unchangedModuleHeader := "## Unchanged Modules\n<details><summary>Expand</summary>\n"
		if _, err := mainStringBuilder.WriteString(fmt.Sprintf("%s\n%s\n</details>\n", unchangedModuleHeader, unchanged)); err != nil {
			return "", err
		}
	}
	return mainStringBuilder.String(), nil
}

func writeUpdatedReferencesTable(
	sb *strings.Builder,
	moduleName string,
	references []*statev1alpha1.ModuleReference,
) error {
	refCount := len(references)
	if _, err := sb.WriteString(fmt.Sprintf(
		"<details><summary>%s: %d update(s)</summary>\n\n| Reference | Manifest Digest |\n|---|---|\n",
		moduleName, refCount,
	)); err != nil {
		return err
	}
	// maxRows maximum amount of reference rows in a table
	const maxRows = 10
	if refCount <= maxRows {
		for _, ref := range references {
			if _, err := sb.WriteString(fmt.Sprintf("| %s | %s |\n", ref.GetName(), ref.GetDigest())); err != nil {
				return err
			}
		}
	} else {
		for i, ref := range references {
			// write only maxRows amount, first 5, last 5
			if (i < maxRows/2) || (i >= refCount-(maxRows/2)) {
				if _, err := sb.WriteString(fmt.Sprintf("| %s | %s |\n", ref.GetName(), ref.GetDigest())); err != nil {
					return err
				}
			}
			if i == maxRows/2 {
				if _, err := sb.WriteString(fmt.Sprintf(
					"| ... %d references skipped ... | ... %d references skipped ... |\n",
					refCount-maxRows, refCount-maxRows,
				)); err != nil {
					return err
				}
			}
		}
	}
	if _, err := sb.WriteString("\n</details>\n"); err != nil {
		return err
	}
	return nil
}
