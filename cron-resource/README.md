# Cron Resource

A Concourse resource for triggering pipelines based on cron expressions.

## Source Configuration

* `expression`: *Required.* The cron expression (e.g., `0 0 * * *` for midnight). Supports seconds if 6 fields are provided.
* `location`: *Optional.* The timezone location (e.g., `America/New_York`). Defaults to `UTC`.

## Behavior

### `check`: Check for new triggers

Scans for timestamps that match the cron expression since the last version.

### `in`: Fetch the trigger timestamp

Writes the triggered timestamp to a file named `timestamp` in the destination directory.

### `out`: Not supported

This resource does not support `put` steps.

## Example Configuration

```yaml
resource_types:
- name: cron
  type: docker-image
  source:
    repository: your-repo/cron-resource

resources:
- name: every-5-minutes
  type: cron
  source:
    expression: "*/5 * * * *"
    location: "Europe/London"

jobs:
- name: periodic-job
  plan:
  - get: every-5-minutes
    trigger: true
  - task: do-something
    config:
      ...
```
