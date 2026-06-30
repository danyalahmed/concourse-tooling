# cPanel Backup to SMB Resource

A Concourse resource that performs cPanel file and database backups over SSH and streams them directly to an SMB share. This approach avoids using local disk space on the Concourse worker, making it ideal for large backups.

## Source Configuration

* `host`: *Required.* SSH hostname of the cPanel server.
* `port`: *Optional.* SSH port (default `22`).
* `username`: *Required.* SSH/cPanel username.
* `ssh_key`: *Required.* Private SSH key for authentication.
* `ssh_key_passphrase`: *Optional.* Passphrase for the SSH key.
* `mysql_password`: *Required.* MySQL password for the cPanel user to perform `mysqldump`.
* `smb_host`: *Required.* SMB server hostname.
* `smb_port`: *Optional.* SMB port (default `445`).
* `smb_username`: *Required.* SMB username.
* `smb_password`: *Required.* SMB password.
* `smb_share`: *Required.* SMB share name.

## Behavior

### `check`: Check for new backups

Always returns a new version based on the current timestamp to ensure backups can be triggered.

### `in`: Perform the backup

1.  Connects to the cPanel server via SSH.
2.  Connects to the SMB share.
3.  Creates a timestamped backup directory on SMB.
4.  Streams each directory in `params.directories` as a `tar.gz` archive directly to SMB.
5.  Streams a full MySQL dump to SMB.
6.  **Verification:** Checks each uploaded file on SMB to ensure it exists and is not empty.
7.  **Retention:** Applies the specified retention policy by deleting old backup directories.

### `out`: Not supported

This resource does not support `put` steps.

## Example Configuration

```yaml
resource_types:
- name: cpanel-backup
  type: registry-image
  source:
    repository: your-repo/cpanel-backup-to-smb-resource

resources:
- name: site-backup
  type: cpanel-backup
  source:
    host: cpanel.example.com
    username: myuser
    ssh_key: ((cpanel-ssh-key))
    mysql_password: ((cpanel-mysql-password))
    smb_host: storage.example.com
    smb_username: ((smb-user))
    smb_password: ((smb-pass))
    smb_share: backups

jobs:
- name: nightly-backup
  plan:
  - get: every-night
    trigger: true
  - get: site-backup
    params:
      parent_dir: "/my-backups"
      directories:
      - "public_html"
      - "mail"
      keep_count: 7
```

## Parameters

* `parent_dir`: *Required.* The base directory on the SMB share where backups will be stored.
* `directories`: *Required (if `db_only` is false).* A list of directories relative to the user's home to back up.
* `excludes`: *Optional.* A list of patterns to exclude from the `tar` archives.
* `db_only`: *Optional.* If `true`, only the MySQL dump will be performed. Defaults to `false`.
* `skip_db`: *Optional.* If `true`, the MySQL dump will be skipped. Defaults to `false`.
* `keep_count`: *Optional.* Number of recent backups to keep.
* `keep_days`: *Optional.* Number of days to keep backups.
