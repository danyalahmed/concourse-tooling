# smb-resource

A [Concourse CI resource](https://concourse-ci.org/resources.html) for transferring files to/from SMB/CIFS network shares.

## Source configuration

```yaml
resource_types:
  - name: smb
    type: registry-image
    source:
      repository: ghcr.io/danyalahmed/smb-resource

resources:
  - name: data
    type: smb
    source:
      host: fileserver.example.com
      share: shared
      username: ((smb-username))
      password: ((smb-password))
      # port: 445 (default)
```

| field | description |
|-------|-------------|
| `host` | SMB server hostname or IP |
| `share` | Share name (optional, default root) |
| `username` | SMB username |
| `password` | SMB password |
| `port` | SMB port (default `445`) |

## Usage

### `get` — download a file

```yaml
- get: data
  params:
    file: backups/db.sql
```

Downloads the file from a versioned remote directory and writes it locally. Metadata includes `filename`, `size_bytes`, `sha256`.

### `put` — upload a file

```yaml
- put: data
  params:
    file: build/output.tgz
```

Creates a timestamped remote directory (`<basename>-<unix>`), uploads the file, and returns the version string.

## Behavior

| command | what happens |
|---------|-------------|
| `check` | Lists directories on the share sorted by modification time. On first run returns only the latest; subsequent runs returns all from the current version forward. |
| `in` | Connects to the share, reads the file specified by `params.file` from the versioned remote directory, streams it to a local file, and returns SHA-256 metadata. |
| `out` | Creates a timestamped remote directory, uploads the local file with SHA-256 hashing, returns version + metadata. |

## Docker

```sh
docker build -t smb-resource .
```

## Local testing

```sh
go build ./cmd/resource/
echo '{"source":{"host":"...","username":"...","password":"...","share":"..."},"params":{"file":"test.txt"}}' | ./resource in /tmp/out
```

## Development

Go 1.26.3. Build: `go build ./cmd/resource/`
