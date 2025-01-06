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
	"encoding/json"
	"fmt"
	"io"

	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
	"go.uber.org/multierr"
)

const ModStateFileName = "state.json"

// ReadModStateFile reads a JSON encoded ModuleState from the given reader before closing it.
func (rw *ReadWriter) ReadModStateFile(readCloser io.ReadCloser) (_ *statev1alpha1.ModuleState, retErr error) {
	defer func() {
		if err := readCloser.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("close file: %w", err))
		}
	}()
	var moduleState statev1alpha1.ModuleState
	if err := json.NewDecoder(readCloser).Decode(&moduleState); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	if err := rw.validator.Validate(&moduleState); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return &moduleState, nil
}

// WriteModStateFile takes a module state and writes it to the given writer before closing it.
func (rw *ReadWriter) WriteModStateFile(writeCloser io.WriteCloser, moduleState *statev1alpha1.ModuleState) (retErr error) {
	defer func() {
		if err := writeCloser.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("close file: %w", err))
		}
	}()
	if err := rw.validator.Validate(moduleState); err != nil {
		return fmt.Errorf("validate: %w", err)
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
