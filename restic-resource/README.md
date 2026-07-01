# Restic Backup Resources for cPanel

This directory contains three Concourse resources for managing cPanel backups using Restic and SMB.

1.  **restic-backup-resource**: Performs backups from cPanel to SMB.
2.  **restic-prune-resource**: Manages retention and prunes the Restic repository.
3.  **restic-restore-resource**: Restores data from SMB back to cPanel.
4.  **restic-stats-resource**: Explores the SMB mount and shows disk usage and file list.

## Common Source Configuration

* `host`: *Required.* SSH hostname of the cPanel server.
* `port`: *Optional.* SSH port (default `22`).
* `username`: *Required.* SSH username.
* `ssh_key`: *Required.* Private SSH key.
* `ssh_key_passphrase`: *Optional.* Passphrase for the SSH key.
* `smb_host`: *Required.* SMB server hostname.
* `smb_username`: *Required.* SMB username.
* `smb_password`: *Required.* SMB password.
* `smb_share`: *Required.* SMB share name.
* `repository_path`: *Optional.* Path within the SMB share where the Restic repository is located.
* `repository_pass`: *Required.* Password for the Restic repository.
* `keep_daily`: *Optional.* Number of daily snapshots to keep (default `7`).
* `keep_weekly`: *Optional.* Number of weekly snapshots to keep (default `4`).
* `keep_monthly`: *Optional.* Number of monthly snapshots to keep (default `12`).
* `keep_yearly`: *Optional.* Number of yearly snapshots to keep (default `3`).

## Resources

### 1. restic-backup-resource

Performs a Restic backup. Mounts cPanel via SSHFS and SMB via CIFS to the Concourse worker.

**`put` Parameters:**

* `directories`: *Optional.* List of directories to backup (relative to home). Defaults to backing up the entire home directory.
* `excludes`: *Optional.* List of patterns to exclude.

### 2. restic-prune-resource

Runs `restic forget --prune` using the retention policy defined in the source configuration.

**`put` Parameters:**

* (None)

### 3. restic-restore-resource

Restores a snapshot from the Restic repository back to the cPanel server.

**`get` Parameters:**

* `snapshot_id`: *Optional.* The ID of the snapshot to restore (default `latest`).
* `target_subdir`: *Optional.* Subdirectory within cPanel to restore to.

### 4. restic-stats-resource

Explores the SMB share to show disk usage and file listings. Useful for debugging and monitoring storage.

**`put` Parameters:**

* (None)

## Requirements

* Concourse worker must run in **privileged** mode to allow mounting filesystems (SSHFS and CIFS).

## Example Pipeline

```yaml
resource_types:
- name: restic-backup
  type: registry-image
  source:
    repository: your-repo/restic-backup-resource
- name: restic-prune
  type: registry-image
  source:
    repository: your-repo/restic-prune-resource

resources:
- name: site-files-restic
  type: restic-backup
  source:
    host: cpanel.example.com
    username: myuser
    ssh_key: ((cpanel-ssh-key))
    smb_host: storage.example.com
    smb_username: ((smb-user))
    smb_password: ((smb-pass))
    smb_share: backups
    repository_path: "/restic-repo"
    repository_pass: ((restic-pass))

jobs:
- name: nightly-backup
  plan:
  - put: site-files-restic
    params:
      directories: ["public_html", "mail"]
  - put: restic-prune # Optional: separate step for pruning
    resource: site-files-restic # Can reuse the same resource definition
    params: { action: "prune" }
```
