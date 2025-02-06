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
	"fmt"
	"io/fs"
	"strconv"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/slogapp"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/modules/private/bufpkg/bufstate"
	"github.com/spf13/pflag"
)

// format is a format to print the casdiff.
type format int

const (
	formatFlagName      = "format"
	formatFlagShortName = "f"

	emptyDiffSameFromAndTo = "empty diff, from and to are the same"
)

const (
	formatText format = iota + 1
	formatMarkdown
)

var (
	formatsValuesToNames = map[format]string{
		formatText:     "text",
		formatMarkdown: "markdown",
	}
	formatsNamesToValues, _ = slicesext.ToUniqueValuesMap(
		slicesext.MapKeysToSlice(formatsValuesToNames),
		func(f format) string { return formatsValuesToNames[f] },
	)
	allFormatsString = slicesext.MapKeysToSortedSlice(formatsNamesToValues)
)

func (f format) String() string {
	if n, ok := formatsValuesToNames[f]; ok {
		return n
	}
	return strconv.Itoa(int(f))
}

func newCommand(name string) *appcmd.Command {
	builder := appext.NewBuilder(
		name,
		appext.BuilderWithLoggerProvider(slogapp.LoggerProvider),
	)
	flags := newFlags()
	return &appcmd.Command{
		Use:       name + " <from> <to>",
		Short:     "Run a CAS diff.",
		Args:      appcmd.ExactArgs(2),
		BindFlags: flags.bind,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
	}
}

type flags struct {
	format string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) bind(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(
		&f.format,
		formatFlagName,
		formatFlagShortName,
		formatText.String(),
		fmt.Sprintf(`The out format to use. Must be one of %s`, allFormatsString),
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	f, ok := formatsNamesToValues[flags.format]
	if !ok {
		return fmt.Errorf("unsupported format %s", flags.format)
	}
	from, to := container.Arg(0), container.Arg(1)
	if from == to {
		return printDiff(newManifestDiff(), f)
	}
	// first, attempt to match from/to as module references in a state file in the same directory
	// where the command is run
	bucket, err := storageos.NewProvider().NewReadWriteBucket(".")
	if err != nil {
		return fmt.Errorf("new rw bucket: %w", err)
	}
	moduleStateReader, err := bucket.Get(ctx, bufstate.ModStateFileName)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("read module state file: %w", err)
		}
		// if the state file does not exist, we assume we are in the cas directory, and that from/to are
		// the manifest paths
		mdiff, err := calculateDiffFromCASDirectory(ctx, bucket, from, to)
		if err != nil {
			return fmt.Errorf("calculate cas diff: %w", err)
		}
		return printDiff(mdiff, f)
	}
	// state file was found, attempt to parse it and match from/to with its references
	stateRW, err := bufstate.NewReadWriter()
	if err != nil {
		return fmt.Errorf("new state rw: %w", err)
	}
	moduleState, err := stateRW.ReadModStateFile(moduleStateReader)
	if err != nil {
		return fmt.Errorf("read module state: %w", err)
	}
	var (
		fromManifestPath string
		toManifestPath   string
	)
	for _, ref := range moduleState.GetReferences() {
		if ref.GetName() == from {
			fromManifestPath = ref.GetDigest()
			if toManifestPath != "" {
				break
			}
		} else if ref.GetName() == to {
			toManifestPath = ref.GetDigest()
			if fromManifestPath != "" {
				break
			}
		}
	}
	if fromManifestPath == "" {
		return fmt.Errorf("from reference %s not found in the module state file", from)
	}
	if toManifestPath == "" {
		return fmt.Errorf("to reference %s not found in the module state file", to)
	}
	if fromManifestPath == toManifestPath {
		return printDiff(newManifestDiff(), f)
	}
	casBucket, err := storageos.NewProvider().NewReadWriteBucket("cas")
	if err != nil {
		return fmt.Errorf("new rw cas bucket: %w", err)
	}
	mdiff, err := calculateDiffFromCASDirectory(ctx, casBucket, fromManifestPath, toManifestPath)
	if err != nil {
		return fmt.Errorf("calculate cas diff from state references: %w", err)
	}
	return printDiff(mdiff, f)
}

// calculateDiffFromCASDirectory takes the cas bucket, and the from/to manifest paths to calculate a
// diff.
func calculateDiffFromCASDirectory(
	ctx context.Context,
	casBucket storage.ReadBucket,
	fromManifestPath string,
	toManifestPath string,
) (*manifestDiff, error) {
	if fromManifestPath == toManifestPath {
		return newManifestDiff(), nil
	}
	fromManifest, err := readManifest(ctx, casBucket, fromManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest from: %w", err)
	}
	toManifest, err := readManifest(ctx, casBucket, toManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest to: %w", err)
	}
	return buildManifestDiff(ctx, fromManifest, toManifest, casBucket)
}

func readManifest(ctx context.Context, bucket storage.ReadBucket, manifestPath string) (bufcas.Manifest, error) {
	data, err := storage.ReadPath(ctx, bucket, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read path: %w", err)
	}
	m, err := bufcas.ParseManifest(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return m, nil
}

func printDiff(mdiff *manifestDiff, f format) error {
	switch f {
	case formatText:
		mdiff.printText()
	case formatMarkdown:
		mdiff.printMarkdown()
	default:
		return fmt.Errorf("format %s not supported", f.String())
	}
	return nil
}
