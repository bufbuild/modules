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
	"fmt"
	"strconv"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/pkg/slogapp"
	"github.com/bufbuild/modules/internal/bufcasdiff"
	"github.com/spf13/pflag"
)

// format is a format to print the casdiff.
type format int

const (
	formatFlagName      = "format"
	formatFlagShortName = "f"
)

const (
	formatText format = iota + 1
	formatMarkdown
)

//nolint:gochecknoglobals // treated as consts
var (
	formatsValuesToNames = map[format]string{
		formatText:     "text",
		formatMarkdown: "markdown",
	}
	formatsNamesToValues, _ = xslices.ToUniqueValuesMap(
		xslices.MapKeysToSlice(formatsValuesToNames),
		func(f format) string { return formatsValuesToNames[f] },
	)
	allFormatsString = xslices.MapKeysToSortedSlice(formatsNamesToValues)
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
	from, to := container.Arg(0), container.Arg(1) //nolint:varnamelen // from/to used symmetrically
	mdiff, err := bufcasdiff.DiffModuleDirectory(ctx, ".", from, to)
	if err != nil {
		return fmt.Errorf("calculate diff: %w", err)
	}
	switch f {
	case formatText:
		fmt.Print(mdiff.String(bufcasdiff.ManifestDiffOutputFormatText))
	case formatMarkdown:
		fmt.Print(mdiff.String(bufcasdiff.ManifestDiffOutputFormatMarkdown))
	default:
		return fmt.Errorf("format %s not supported", f.String())
	}
	return nil
}
