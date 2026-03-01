# SignalForge

SignalForge is a standalone workflow monorepo. It contains reusable workflow implementations that run inside sandbox environments and are invoked by `control-plane` via image reference.

SignalForge does not host MCP tool sidecars. Workflows in this repo can call tools from a separate tool repo through configured MCP endpoints.

## Repository layout

```text
SignalForge/
├── workflows/
│   └── <workflow-name>/
│       ├── main.go
│       ├── go.mod
│       ├── Dockerfile
│       └── workflow.yaml
├── shared/
├── spec/
├── docs/
├── .github/
│   ├── actions/
│   └── workflows/
├── release-please-config.json
└── .release-please-manifest.json
```

## CI/CD behavior

- Pull requests:
  - detect changed `workflows/<name>` directories
  - run lint/test for changed workflows
  - build Docker images tagged `sha-<commit>`
- Push to `main`:
  - run release-please and open/update release PRs for changed workflows
- Release tags:
  - build and push versioned workflow images

## Versioning model

- Each `workflows/<name>` directory is an independent release-please component.
- Release tags follow `<workflow>-v<semver>` (for example `echo-v1.2.3`).
- Production deploys should pin semver or digest, not `latest`.
