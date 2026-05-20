# sdk

A generic Concourse resource engine that eliminates boilerplate for `check`, `in`, and `out` commands.

## Usage

Implement the `Driver` interface with your resource's types and logic, then call `RunCommand`:

```go
sdk.RunCommand(&MyDriver{})
```

The SDK handles stdin/stdout JSON encoding, argument parsing, and error reporting.

## Interface

```go
type Driver[Source, Version, InParams, OutParams, Metadata any] interface {
    Check(ctx, source, version) ([]Version, error)
    In(ctx, source, version, params, targetDir) (Version, Metadata, error)
    Out(ctx, source, params, sourceDir) (Version, Metadata, error)
}
```

Zero external dependencies.
