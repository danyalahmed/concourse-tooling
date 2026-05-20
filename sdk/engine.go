package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Driver[Source any, Version any, InParams any, OutParams any, Metadata any] interface {
	Check(ctx context.Context, source Source, version *Version) ([]Version, error)
	In(ctx context.Context, source Source, version Version, params InParams, targetDir string) (Version, Metadata, error)
	Out(ctx context.Context, source Source, params OutParams, sourceDir string) (Version, Metadata, error)
}

type Request[Source any, Version any, Params any] struct {
	Source  Source  `json:"source"`
	Version Version `json:"version,omitempty"`
	Params  Params  `json:"params,omitempty"`
}

type Response[Version any, Metadata any] struct {
	Version  Version  `json:"version,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

func RunCommand[Source any, Version any, InParams any, OutParams any, Metadata any](driver Driver[Source, Version, InParams, OutParams, Metadata]) {
	ctx := context.Background()

	command := filepath.Base(os.Args[0])

	switch command {
	case "check":
		var req Request[Source, Version, any]
		mustDecode(&req)

		versions, err := driver.Check(ctx, req.Source, &req.Version)
		if err != nil {
			fail("Driver check execution failed: %v", err)
		}
		// concourse expects an array of versions, even if zero version is returned
		if versions == nil {
			versions = []Version{}
		}
		mustEncode(versions)
	case "in":
		if len(os.Args) < 2 {
			fail("Missing target directory argument for 'in' command")
		}
		targetDir := os.Args[1]

		var req Request[Source, Version, InParams]
		mustDecode(&req)

		version, metadata, err := driver.In(ctx, req.Source, req.Version, req.Params, targetDir)
		if err != nil {
			fail("Driver in execution failed: %v", err)
		}

		resp := Response[Version, Metadata]{
			Version:  version,
			Metadata: metadata,
		}
		mustEncode(resp)
	case "out":
		sourceDir := os.Args[1]

		var req Request[Source, any, OutParams]
		mustDecode(&req)

		version, metadata, err := driver.Out(ctx, req.Source, req.Params, sourceDir)
		if err != nil {
			fail("Driver out execution failed: %v", err)
		}

		resp := Response[Version, Metadata]{
			Version:  version,
			Metadata: metadata,
		}
		mustEncode(resp)
	default:
		fail("Unknown command: %s", command)
	}
}

func mustDecode(v any) {
	if err := json.NewDecoder(os.Stdin).Decode(v); err != nil {
		fail("failed to decode request: %v", err)
	}
}

func mustEncode(v any) {
	if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
		if errors.Is(err, io.EOF) {
			fail("standard input stream was completely empty")
		}
		fail("failed to encode response: %v", err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
