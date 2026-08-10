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

// LatestModuleDigest returns the digest of the last reference appended to the
// module state, or an empty string if the module has no state file yet or its
// state file has no references. It assumes the same sync dir structure as
// AppendModuleReference.
func (rw *ReadWriter) LatestModuleDigest(
	rootSyncDir string,
	ownerName string,
	repoName string,
) (string, error) {
	modState, err := rw.readModuleState(filepath.Join(rootSyncDir, ownerName, repoName, ModStateFileName))
	if err != nil {
		return "", err
	}
	references := modState.GetReferences()
	if len(references) == 0 {
		return "", nil
	}
	return references[len(references)-1].GetDigest(), nil
}

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
	modState, err := rw.readModuleState(modFilePath)
	if err != nil {
		return err
	}
	modState.SetReferences(append(modState.GetReferences(), statev1alpha1.ModuleReference_builder{Name: reference, Digest: digest}.Build()))
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
			globalState.GetModules()[i].SetLatestReference(reference)
			break
		}
	}
	if !found {
		globalState.SetModules(append(
			globalState.GetModules(),
			statev1alpha1.GlobalStateReference_builder{
				ModuleName:      moduleName,
				LatestReference: reference,
			}.Build(),
		))
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

// readModuleState reads the module state file at the given path, returning an
// empty state if the file does not exist yet.
func (rw *ReadWriter) readModuleState(modFilePath string) (*statev1alpha1.ModuleState, error) {
	if _, err := os.Stat(modFilePath); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat file: %w", err)
		}
		return &statev1alpha1.ModuleState{}, nil
	}
	modStateFile, err := os.Open(modFilePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	modState, err := rw.ReadModStateFile(modStateFile)
	if err != nil {
		return nil, fmt.Errorf("read module state file: %w", err)
	}
	return modState, nil
}
