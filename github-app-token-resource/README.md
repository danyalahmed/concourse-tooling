# github-app-token

A [Concourse CI resource](https://concourse-ci.org/resources.html) that generates GitHub App installation access tokens.

## Source configuration

```yaml
resource_types:
  - name: github-app-token
    type: registry-image
    source:
      repository: ghcr.io/danyalahmed/github-app-token

resources:
  - name: github-token
    type: github-app-token
    source:
      app_id: "<your GitHub App ID>"
      installation_id: "<installation ID>"
      private_key: |
        -----BEGIN RSA PRIVATE KEY-----
        ...
        -----END RSA PRIVATE KEY-----
```

| field | description |
|-------|-------------|
| `app_id` | GitHub App ID (numeric). Found in your app settings. |
| `installation_id` | Installation ID of the app on the target org/user. |
| `private_key` | Private key generated in your GitHub App settings. PEM-encoded RSA key. |

Store `private_key` via your credential manager — never inline it:
```yaml
source:
  private_key: ((github-app-private-key))
```

## Usage

### `get` step

Fetches a fresh installation access token and writes it to `token`:

```yaml
jobs:
  - name: deploy
    plan:
      - get: github-token
      - task: use-token
        file: ci/tasks/deploy.yml
```

The token file is at `github-token/token` (permissions `0o600`). Use it in subsequent tasks:

```yaml
# ci/tasks/deploy.yml
platform: linux
inputs:
  - name: github-token
run:
  path: sh
  args:
    - -c
    - |
      TOKEN=$(cat github-token/token)
      curl -H "Authorization: Bearer $TOKEN" ...
```

The token expires after 1 hour (GitHub-imposed). Use a new `get` step to refresh.

### `put` step

No-op. This resource only generates tokens; it doesn't push anything.

## Behavior

| command | what happens |
|---------|-------------|
| `check` | Returns current Unix timestamp as the version. Each `get` produces a fresh token. |
| `in` | Generates a JWT signed with `private_key`, exchanges it via `POST /app/installations/{id}/access_tokens` for an installation token, writes the token to `dest/token` (`0o600`). |
| `out` | No-op. Returns `{"version": {"version": "noop"}}`. |

- GitHub API timeout: 10s
- JWT expiry: 10 min (sufficient for the API exchange)
- Installation access token expiry: 1 hour (set by GitHub)

## Docker

```sh
docker build -t github-app-token-resource .
```

## Security

- The access token is written to `token` with `0o600` permissions.
- The token is never included in the resource metadata (would leak into build output).
- The private key is only held in memory during the `in` invocation.
- Use Concourse credential managers (`((vars))`) for `private_key` — never embed it in pipeline YAML.

## Local testing

```sh
go build ./cmd/resource/
echo '{"source":{"app_id":"...","installation_id":"...","private_key":"..."}}' | ./resource in /tmp/token
cat /tmp/token
```

## Development

- Devcontainer: Go 1.26-alpine, gopls with staticcheck, `gofmt` on save.
- Build: `go build ./cmd/resource/`
- Tests: `go test ./internal/resource/...`
