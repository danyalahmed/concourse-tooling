# cPanel Database Backup Resource

A Concourse resource that performs cPanel database backups over SSH and streams them directly to an SMB share as gzipped SQL files.

## Source Configuration

* `host`: *Required.* SSH hostname of the cPanel server.
* `port`: *Optional.* SSH port (default `22`).
* `username`: *Required.* SSH username and default MySQL username.
* `ssh_key`: *Required.* Private SSH key for authentication.
* `ssh_key_passphrase`: *Optional.* Passphrase for the SSH key.
* `admin_mysql_password`: *Required.* MySQL password for the default user or admin.
* `smb_host`: *Required.* SMB server hostname.
* `smb_port`: *Optional.* SMB port (default `445`).
* `smb_username`: *Required.* SMB username.
* `smb_password`: *Required.* SMB password.
* `smb_share`: *Required.* SMB share name.
* `keep_daily`: *Optional.* Number of daily backups to keep (default `7`).
* `keep_weekly`: *Optional.* Number of weekly backups to keep (default `4`).
* `keep_monthly`: *Optional.* Number of monthly backups to keep (default `12`).
* `keep_yearly`: *Optional.* Number of yearly backups to keep (default `3`).

## Behavior

### `check`: Check for new backups

Always returns a new version based on the current timestamp.

### `out`: Perform the backup

1.  Connects to the cPanel server via SSH.
2.  Connects to the SMB share.
3.  Creates a date-stamped backup directory on SMB (e.g., `parent_dir/2023-10-27`).
4.  If `all_dbs` is true, it fetches the list of all databases.
5.  Iterates through databases and streams `mysqldump | gzip` directly to SMB.
6.  Uses specific credentials from the `databases` map if provided, otherwise falls back to `admin_mysql_password`.
7.  Applies GFS retention policy by deleting old backup directories.

### `in`: No-op

Returns the provided version.

## Example Configuration

```yaml
resource_types:
- name: cpanel-db-backup
  type: registry-image
  source:
    repository: your-repo/cpanel-db-backup-resource

resources:
- name: db-backup
  type: cpanel-db-backup
  source:
    host: cpanel.example.com
    username: myuser
    ssh_key: ((cpanel-ssh-key))
    admin_mysql_password: ((cpanel-admin-mysql-password))
    smb_host: storage.example.com
    smb_username: ((smb-user))
    smb_password: ((smb-pass))
    smb_share: backups

jobs:
- name: nightly-db-backup
  plan:
  - put: db-backup
    params:
      parent_dir: "/db-backups"
      all_dbs: true
      databases:
        wp_site1:
          user: site1_user
          pass: ((site1-pass))
```

## Parameters

* `parent_dir`: *Required.* The base directory on the SMB share.
* `all_dbs`: *Optional.* If `true`, backup all databases accessible by the admin user. Defaults to `false`.
* `databases`: *Optional.* A map of database names to credentials.
    * `user`: *Optional.* MySQL username for this database.
    * `pass`: *Optional.* MySQL password for this database.
