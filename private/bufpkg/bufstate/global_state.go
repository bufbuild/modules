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

package bufstate

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	modulestatev1alpha1 "github.com/bufbuild/modules/private/gen/modulestate/v1alpha1"
	"go.uber.org/multierr"
)

const GlobalStateFileName = "state.json"

// ReadGlobalState reads a JSON encoded GlobalState from the given reader before closing it.
func ReadGlobalState(reader io.ReadCloser) (_ *modulestatev1alpha1.GlobalState, retErr error) {
	defer func() {
		if err := reader.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("close file: %w", err))
		}
	}()
	var s modulestatev1alpha1.GlobalState
	if err := json.NewDecoder(reader).Decode(&s); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if err := validateGlobalState(&s); err != nil {
		return nil, fmt.Errorf("invalid global state: %w", err)
	}
	return &s, nil
}

// WriteGlobalState takes a global state and writes it to the given writer before closing it.
func WriteGlobalState(writer io.WriteCloser, s *modulestatev1alpha1.GlobalState) (retErr error) {
	defer func() {
		if err := writer.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("close file: %w", err))
		}
	}()
	if err := validateGlobalState(s); err != nil {
		return fmt.Errorf("invalid global state: %w", err)
	}
	mods := s.GetModules()
	sort.Slice(mods, func(i, j int) bool {
		return mods[i].GetModuleName() < mods[j].GetModuleName()
	})
	s.Modules = mods
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal state: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("write to file: %w", err)
	}
	return nil
}

// validateGlobalState checks there are no repeated modules or empty values in names or references.
func validateGlobalState(s *modulestatev1alpha1.GlobalState) error {
	mods := make(map[string]struct{}, len(s.GetModules()))
	for _, mod := range s.GetModules() {
		if len(mod.GetModuleName()) == 0 {
			return fmt.Errorf("empty module name with digest %q", mod.GetLatestReference())
		}
		if len(mod.GetLatestReference()) == 0 {
			return fmt.Errorf("module %q has an empty latest reference", mod.GetModuleName())
		}
		if _, alreadyExists := mods[mod.GetModuleName()]; alreadyExists {
			return fmt.Errorf("module %q appears multiple times", mod.GetModuleName())
		}
		mods[mod.GetModuleName()] = struct{}{}
	}
	return nil
}
