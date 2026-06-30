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
      # watch: nuxt-test            # path to monitor for changes (optional)
```

| field | description |
|-------|-------------|
| `host` | SMB server hostname or IP |
| `share` | Share name (optional, default root) |
| `username` | SMB username |
| `password` | SMB password |
| `port` | SMB port (default `445`) |
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

## Local testing

```sh
go build ./cmd/resource/

# download a single file
echo '{"source":{"host":"...","username":"...","password":"...","share":"..."},"params":{"file":"test.txt"}}' | ./resource in /tmp/out

# download a directory recursively
echo '{"source":{...},"params":{"file":"backups/"}}' | ./resource in /tmp/out

# upload a file
echo '{"source":{...},"params":{"file":"build.tgz","dest":"remote/build.tgz"}}' | ./resource out /tmp/source

# upload a directory recursively
echo '{"source":{...},"params":{"file":"dist/","dest":"releases/v1.0/"}}' | ./resource out /tmp/source
```

## Development

Go 1.26.4. Build: `go build ./cmd/resource/`
