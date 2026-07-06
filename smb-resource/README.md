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
      smb_host: fileserver.example.com
      smb_share: shared
      smb_username: ((smb-username))
      smb_password: ((smb-password))
      # smb_port: 445 (default)
      # watch: nuxt-test            # path to monitor for changes (optional)
```

| field | description |
|-------|-------------|
| `smb_host` | SMB server hostname or IP |
| `smb_share` | Share name (optional, default root) |
| `smb_username` | SMB username |
| `smb_password` | SMB password |
| `smb_port` | SMB port (default `445`) |
| `watch` | Path on the share to monitor for changes (optional). When set, `check` recursively scans this path and returns a new version whenever any file or directory within it is modified. |

## Usage

### `get` — download a file or directory

```yaml
- get: data
  params:
    file: backups/db.sql
```

Downloads a single file from the share to `db.sql` locally. Metadata includes `filename`, `size_bytes`, `sha256`.

If `params.file` points to a directory, the entire directory tree is downloaded recursively, preserving the structure:

```yaml
- get: data
  params:
    file: project/src/
```

### `put` — upload a file or directory

```yaml
- put: data
  params:
    file: build/output.tgz
```

Uploads a single file to the share. `params.dest` optionally specifies the remote path (defaults to the basename of `params.file`). Parent directories on the remote side are created automatically.

If `params.file` points to a local directory, the entire directory tree is uploaded recursively, preserving the structure:

```yaml
- put: data
  params:
    file: dist/
    dest: releases/v1.0/
```

### Exploration (Experimental)

If `put` is called without a `file` parameter, the resource will attempt to mount the SMB share using system commands and list the contents. This requires the worker to run in **privileged** mode.

```yaml
- put: data
  params: {}
```

## Behavior

| command | what happens |
|---------|-------------|
| `check` | If `source.watch` is set, recursively scans that path and returns a new version when any file or directory changes. Otherwise scans the share root. On first run returns the latest mod time; subsequent runs return a new version only when the mod time differs. |
| `in` | Connects to the share, reads the file or directory specified by `params.file`, streams to the local target. Returns SHA-256 metadata for single files. |
| `out` | Uploads a local file or directory to the share. Returns version + metadata. Remote parent directories are created as needed. |

## Docker

```sh
docker build -t smb-resource .
```

## Development

Go 1.26.4. Build: `go build ./cmd/resource/`
