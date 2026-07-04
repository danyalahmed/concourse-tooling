# cPanel Database Backup Resource

A Concourse resource that performs database dumps (MySQL or PostgreSQL) on a cPanel server via SSH. The dumps are stored uncompressed in a local directory on the cPanel server, intended to be subsequently picked up by a backup tool like Restic.

## Source Configuration

* `host`: *Required.* SSH hostname of the cPanel server.
* `port`: *Optional.* SSH port (default `22`).
* `username`: *Required.* SSH username.
* `ssh_key`: *Required.* Private SSH key for authentication.
* `ssh_key_passphrase`: *Optional.* Passphrase for the SSH key.

## Behavior

### `check`: Check for new backups

Always returns a new version based on the current timestamp.

### `out`: Perform the dump

1.  Connects to the cPanel server via SSH.
2.  Ensures the backup directory exists: `/home/{ssh_username}/database-dumps/{engine}/`.
3.  Iterates through specified databases and executes `mysqldump` or `pg_dump` to create uncompressed `.sql` files.
4.  Database files are named `{database-name}.sql` (no timestamps in filenames).

### `in`: No-op

Returns the provided version.

## Example Configuration

```yaml
resource_types:
- name: cpanel-db-dump
  type: registry-image
  source:
    repository: your-repo/cpanel-db-backup-resource

resources:
- name: db-dump
  type: cpanel-db-dump
  source:
    host: cpanel.example.com
    username: myuser
    ssh_key: ((cpanel-ssh-key))

jobs:
- name: nightly-db-backup
  plan:
  - put: db-dump
    params:
      engine: postgres
      db_user: postgres_admin
      db_pass: ((postgres-pass))
      databases:
        - site1_db
        - site2_db
```

## Parameters

* `engine`: *Optional.* The database engine to use (`mysql` or `postgres`). Defaults to `mysql`.
* `db_user`: *Required.* Database admin username.
* `db_pass`: *Required.* Database admin password.
* `databases`: *Required.* A list of database names to dump.
