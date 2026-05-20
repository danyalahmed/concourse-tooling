package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

// Metadata represents Concourse resource metadata.
type Metadata []MetadataItem

// MetadataItem represents a single metadata entry.
type MetadataItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Driver defines the interface for a Concourse resource.
type Driver[Source any, Version any, InParams any, OutParams any] interface {
	Check(ctx context.Context, source Source, version *Version) ([]Version, error)
	In(ctx context.Context, source Source, version Version, params InParams, targetDir string) (Version, Metadata, error)
	Out(ctx context.Context, source Source, params OutParams, sourceDir string) (Version, Metadata, error)
}

type Request[Source any, Version any, Params any] struct {
	Source  Source  `json:"source"`
	Version Version `json:"version,omitempty"`
	Params  Params  `json:"params,omitempty"`
}

type Response[Version any] struct {
	Version  Version  `json:"version,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

// RunCommand is the entry point for the resource commands.
func RunCommand[Source any, Version any, InParams any, OutParams any](driver Driver[Source, Version, InParams, OutParams]) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	command := filepath.Base(os.Args[0])

	switch command {
	case "check":
		var req Request[Source, Version, any]
		mustDecode(&req)

		versions, err := driver.Check(ctx, req.Source, &req.Version)
		if err != nil {
			fail("Driver check execution failed: %v", err)
		}
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

		resp := Response[Version]{
			Version:  version,
			Metadata: metadata,
		}
		mustEncode(resp)
	case "out":
		if len(os.Args) < 2 {
			fail("Missing source directory argument for 'out' command")
		}
		sourceDir := os.Args[1]

		var req Request[Source, any, OutParams]
		mustDecode(&req)

		version, metadata, err := driver.Out(ctx, req.Source, req.Params, sourceDir)
		if err != nil {
			fail("Driver out execution failed: %v", err)
		}

		resp := Response[Version]{
			Version:  version,
			Metadata: metadata,
		}
		mustEncode(resp)
	default:
		fail("Unknown command: %s", command)
	}
}

// Log writes a message to stderr for Concourse to display.
func Log(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
}

// Logf writes a formatted message to stderr for Concourse to display.
func Logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
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
	Logf(format, args...)
	os.Exit(1)
}

