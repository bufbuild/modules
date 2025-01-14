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
	"io"
	"sort"

	"github.com/bufbuild/buf/private/pkg/protoencoding"
	statev1alpha1 "github.com/bufbuild/modules/private/gen/modules/state/v1alpha1"
	"go.uber.org/multierr"
)

const GlobalStateFileName = "state.json"

// ReadGlobalState reads a JSON encoded GlobalState from the given reader before closing it.
func (rw *ReadWriter) ReadGlobalState(reader io.ReadCloser) (_ *statev1alpha1.GlobalState, retErr error) {
	defer func() {
		if err := reader.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("close file: %w", err))
		}
	}()
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read global state file: %w", err)
	}
	var globalState statev1alpha1.GlobalState
	if err := protoencoding.NewJSONUnmarshaler(protoencoding.EmptyResolver).Unmarshal(bytes, &globalState); err != nil {
		return nil, fmt.Errorf("unmarshal global state: %w", err)
	}
	if err := rw.validator.Validate(&globalState); err != nil {
		return nil, fmt.Errorf("validate global state: %w", err)
	}
	return &globalState, nil
}

// WriteGlobalState takes a global state and writes it to the given writer before closing it.
func (rw *ReadWriter) WriteGlobalState(writer io.WriteCloser, globalState *statev1alpha1.GlobalState) (retErr error) {
	defer func() {
		if err := writer.Close(); err != nil {
			retErr = multierr.Append(retErr, fmt.Errorf("close file: %w", err))
		}
	}()
	if err := rw.validator.Validate(globalState); err != nil {
		return fmt.Errorf("validate global state: %w", err)
	}
	mods := globalState.GetModules()
	sort.Slice(mods, func(i, j int) bool {
		return mods[i].GetModuleName() < mods[j].GetModuleName()
	})
	globalState.SetModules(mods)
	data, err := protoencoding.NewJSONMarshaler(
		protoencoding.EmptyResolver,
		protoencoding.JSONMarshalerWithUseProtoNames(),
		protoencoding.JSONMarshalerWithIndent(),
	).Marshal(globalState)
	if err != nil {
		return fmt.Errorf("marshal global state: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("write to file: %w", err)
	}
	return nil
}
