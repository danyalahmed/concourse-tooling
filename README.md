# concourse-tooling

Go-based [Concourse CI](https://concourse-ci.org) resource toolkit. A multi-module workspace with a shared SDK and ready-to-use resources.

| module | description |
|--------|-------------|
| [sdk](./sdk) | Generic resource engine — boilerplate for `check`/`in`/`out` command dispatch |
| [github-app-token](./github-app-token-resource) | GitHub App installation access token generator |
| [smb-resource](./smb-resource) | SMB/CIFS file download/upload resource |

## Development

Go 1.26.3 workspace. Build everything with:

```sh
go build ./...
```

Devcontainer includes gopls with staticcheck and gofmt-on-save.
