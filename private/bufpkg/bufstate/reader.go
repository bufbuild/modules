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

	"buf.build/go/protovalidate"
)

type ReadWriter struct {
	validator protovalidate.Validator
}

func NewReadWriter() (*ReadWriter, error) {
	v, err := protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("initialize validator: %w", err)
	}
	return &ReadWriter{validator: v}, nil
}
