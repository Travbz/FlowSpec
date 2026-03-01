# FlowSpec Workflow Spec

FlowSpec stores workflow implementations and workflow metadata.

It does not store MCP tool-sidecar implementations. MCP tools live in a separate repository and are provided to workflows at runtime by sandbox configuration.

## Workflow Directory Contract

Each workflow directory under `workflows/<name>/` should contain:

- `main.go` (or language equivalent runtime entrypoint)
- `go.mod` or language package manifest
- `Dockerfile`
- `workflow.yaml`

## `workflow.yaml` Purpose

`workflow.yaml` defines workflow metadata used by operators and automation:

- identity (`name`, `version`)
- runtime metadata (`language`, `entrypoint`)
- I/O contract shape
- container image metadata

## Runtime Behavior

- `CommandGrid` resolves workflow image and launches sandbox runtime.
- workflow process executes inside sandbox.
- workflow may call external MCP tools through configured endpoints.
- workflow LLM calls route through the configured proxy stack.

## Release Model

- Each `workflows/<name>` directory is versioned independently.
- release-please creates component tags such as `<name>-vX.Y.Z`.
- image publish pipeline builds versioned images from release tags.
