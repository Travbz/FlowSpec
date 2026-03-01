# FlowSpec

FlowSpec is a standalone workflow monorepo. It contains reusable workflow implementations that run inside sandbox environments and are invoked by `CommandGrid` via image reference.

FlowSpec does not host MCP tool sidecars. Workflows in this repo can call tools from a separate tool repo through configured MCP endpoints.

## Workflow integration policy (hybrid)

FlowSpec uses a hybrid integration model:

- Workflows may call external APIs directly when the integration is simple, low-risk, and only used by one workflow.
- Shared or high-risk integrations should be implemented as reusable tools in `ToolCore`.

Promote a direct workflow integration into `ToolCore` when any of these are true:

- The integration is reused by 2+ workflows.
- The integration handles high-value secrets or sensitive data.
- Centralized policy enforcement is required (validation, audit, retries, rate limits, allowlists).

## Repository layout

```text
FlowSpec/
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
