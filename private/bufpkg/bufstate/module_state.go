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

	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
	"go.uber.org/multierr"
)

const ModStateFileName = "state.json"

// ReadModStateFile reads a JSON encoded ModuleState from the given reader before closing it.
func ReadModStateFile(readCloser io.ReadCloser) (_ *statev1alpha1.ModuleState, retErr error) {
	defer func() {
		if err := readCloser.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("close file: %w", err))
		}
	}()
	var moduleState statev1alpha1.ModuleState
	if err := json.NewDecoder(readCloser).Decode(&moduleState); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if err := validateModuleState(&moduleState); err != nil {
		return nil, fmt.Errorf("invalid module state: %w", err)
	}
	return &moduleState, nil
}

// WriteModStateFile takes a module state and writes it to the given writer before closing it.
func WriteModStateFile(writeCloser io.WriteCloser, moduleState *statev1alpha1.ModuleState) (retErr error) {
	defer func() {
		if err := writeCloser.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("close file: %w", err))
		}
	}()
	if err := validateModuleState(moduleState); err != nil {
		return fmt.Errorf("invalid module state: %w", err)
	}
	data, err := json.MarshalIndent(moduleState, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal state: %w", err)
	}

	if _, err := writeCloser.Write(data); err != nil {
		return fmt.Errorf("write to file: %w", err)
	}
	return nil
}

// validateModuleState checks there are no repeated reference names or empty values in references or
// digests.
func validateModuleState(s *statev1alpha1.ModuleState) error {
	refs := make(map[string]struct{}, len(s.GetReferences()))
	for _, ref := range s.GetReferences() {
		if len(ref.GetName()) == 0 {
			return fmt.Errorf("empty reference name with digest %q", ref.GetDigest())
		}
		if len(ref.GetDigest()) == 0 {
			return fmt.Errorf("reference %q has an empty digest", ref.GetName())
		}
		if _, alreadyExists := refs[ref.GetName()]; alreadyExists {
			return fmt.Errorf("reference %q appears multiple times", ref.GetName())
		}
		refs[ref.GetName()] = struct{}{}
	}
	return nil
}
