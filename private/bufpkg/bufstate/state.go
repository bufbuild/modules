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

package bufstate

import (
	"fmt"
	"os"
	"path/filepath"

	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
)

const SyncRoot = "modules/sync"

// AppendModuleReference appends a reference-digest pair at the end of the module
// state, and updates the module's latest reference in the global state. It
// assumes the structure of the sync dir is
// `root-sync-dir/owner-name/repo-name/state.json` for module state file, and
// `root-sync-dir/state.json` for global state file.
func (rw *ReadWriter) AppendModuleReference(
	rootSyncDir string,
	ownerName string,
	repoName string,
	reference string,
	digest string,
) error {
	modFilePath := filepath.Join(rootSyncDir, ownerName, repoName, ModStateFileName)
	var modState *statev1alpha1.ModuleState
	if _, err := os.Stat(modFilePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat file: %w", err)
		}
		modState = &statev1alpha1.ModuleState{}
	} else {
		modStateFile, err := os.Open(modFilePath)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		modState, err = rw.ReadModStateFile(modStateFile)
		if err != nil {
			return fmt.Errorf("read module state file: %w", err)
		}
	}
	modState.References = append(modState.GetReferences(), &statev1alpha1.ModuleReference{Name: reference, Digest: digest})
	// As the state file read/write functions both close after their operations,
	// we need to re-open another io.WriteCloser here, the easiest way is to
	// truncate the file with Create if it exists.
	modStateFile, err := os.Create(modFilePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	if err := rw.WriteModStateFile(modStateFile, modState); err != nil {
		return fmt.Errorf("write module state file: %w", err)
	}

	globalFilePath := filepath.Join(rootSyncDir, GlobalStateFileName)
	var globalState *statev1alpha1.GlobalState
	if _, err := os.Stat(globalFilePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat file: %w", err)
		}
		globalState = &statev1alpha1.GlobalState{}
	} else {
		globalStateFile, err := os.Open(globalFilePath)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		globalState, err = rw.ReadGlobalState(globalStateFile)
		if err != nil {
			return fmt.Errorf("read module state file: %w", err)
		}
	}
	moduleName := filepath.Join(ownerName, repoName)
	var found bool
	for i := range len(globalState.GetModules()) {
		if globalState.GetModules()[i].GetModuleName() == moduleName {
			found = true
			globalState.GetModules()[i].LatestReference = reference
			break
		}
	}
	if !found {
		globalState.Modules = append(
			globalState.GetModules(),
			&statev1alpha1.GlobalStateReference{
				ModuleName:      moduleName,
				LatestReference: reference,
			},
		)
	}
	// As the state file read/write functions both close after their operations,
	// we need to re-open another io.WriteCloser here, the easiest way is to
	// truncate the file with Create if it exists.
	globalStateFile, err := os.Create(globalFilePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	if err := rw.WriteGlobalState(globalStateFile, globalState); err != nil {
		return fmt.Errorf("write global state file: %w", err)
	}
	return nil
}
