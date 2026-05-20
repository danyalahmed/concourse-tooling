# cpanel-backup-to-smb-resource

A [Concourse CI resource](https://concourse-ci.org/resources.html) that streams cPanel backups directly to an SMB/CIFS share — no intermediate disk on the Concourse runner.

Connects to a cPanel server via SSH, streams configured directories (as `tar.gz`) and a `mysqldump --all-databases` export directly to SMB.

## Source configuration

```yaml
resource_types:
  - name: cpanel-backup
    type: registry-image
    source:
      repository: ghcr.io/danyalahmed/cpanel-backup-to-smb-resource

resources:
  - name: nightly
    type: cpanel-backup
    source:
      host: cpanel.example.com
      username: nuxt
      ssh_key: ((cpanel-ssh-key))
      # ssh_key_passphrase: ((cpanel-ssh-passphrase))   # if key is encrypted
      mysql_password: ((cpanel-mysql-password))
      smb_host: fileserver.example.com
      smb_share: backups
      smb_username: ((smb-username))
      smb_password: ((smb-password))
      # ssh_port: 22                (default)
      # smb_port: 445               (default)
```

| field | description |
|-------|-------------|
| `host` | cPanel server hostname or IP |
| `username` | cPanel SSH / MySQL username |
| `ssh_key` | PEM-encoded SSH private key (RSA, ECDSA, or Ed25519) |
| `ssh_key_passphrase` | Passphrase for an encrypted SSH key (optional) |
| `mysql_password` | MySQL password for `mysqldump` |
| `smb_host` | SMB server hostname or IP |
| `smb_share` | SMB share name |
| `smb_username` | SMB username |
| `smb_password` | SMB password |
| `ssh_port` | SSH port (default `22`) |
| `smb_port` | SMB port (default `445`) |

## Usage

### `get` — run a backup

```yaml
- get: nightly
  params:
    parent_dir: nuxt-prod
    directories:
      - public_html
      - zikr_api
    excludes:
      - node_modules
      - public_html/cache
```

| param | description |
|-------|-------------|
| `parent_dir` | Root directory on SMB under which backup snapshots are created |
| `directories` | List of remote directory paths (relative to `/home/<user>/`) to back up |
| `excludes` | Exclude patterns (optional). Simple names match any directory; `dir/sub` only applies when `dir` is in the list. Excluding an entire directory skips it. |

Each backup run produces a timestamped subdirectory on SMB:

```
<parent_dir>/
  backup_2026-05-20_14-30-00/
    files/
      public_html.tar.gz
      zikr_api.tar.gz
    database/
      all_dbs_2026-05-20.sql
```

Files are streamed via `tar -czf -` on the remote host; databases via `mysqldump --all-databases`. Neither touches the Concourse runner's disk.

### `put` — no-op

```yaml
- put: nightly
```

Returns version `"noop"` with no metadata. The `put` step is a no-op (restore not yet implemented).

## Behavior

| command | what happens |
|---------|-------------|
| `check` | Returns the current Unix timestamp as a new version. |
| `in` | Connects to cPanel via SSH and to SMB. Streams each configured directory as a compressed tarball and all databases as a SQL dump to SMB. Version is the Unix timestamp of the run. |
| `out` | No-op (`"noop"`). |

## Docker

Build from the repo root (required by `go.work`):

```sh
docker build -t cpanel-backup-to-smb-resource -f cpanel-backup-to-smb-resource/Dockerfile .
```

## Local testing

```sh
go build ./cpanel-backup-to-smb-resource/cmd/resource/

# run a backup (in)
echo '{"source":{"host":"...","username":"...","ssh_key":"...","ssh_key_passphrase":"...","mysql_password":"...","smb_host":"...","smb_share":"...","smb_username":"...","smb_password":"..."},"params":{"parent_dir":"test","directories":["public_html"],"excludes":["node_modules"]}}' \
  | ./resource in /tmp/out

# check
echo '{"source":{...}}' | ./resource check

# no-op out
echo '{"source":{...},"params":{}}' | ./resource out /tmp/source
```

## Development

Go 1.26.3. Build: `go build ./cpanel-backup-to-smb-resource/cmd/resource/`
