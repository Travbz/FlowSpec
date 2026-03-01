# CommandGrid Integration

This document defines the boundary between `CommandGrid`, FlowSpec workflows, and the separate MCP tool repository.

## Source-of-Truth Boundaries

- FlowSpec:
  - workflow code
  - workflow metadata (`workflow.yaml`)
  - workflow CI/release automation
- Tool repository:
  - MCP tool-sidecar implementations
  - tool-specific release lifecycle
- CommandGrid:
  - orchestration and sandbox lifecycle
  - secret/session injection
  - workflow image selection and execution

## Invocation Contract

CommandGrid should invoke workflow images by immutable identity:

- `workflow_slug` (for example `echo`)
- `version` (semver tag)
- optional image digest

Recommended image pattern:

- `ghcr.io/<org>/workflow-<workflow_slug>:<version>`

Preferred production pattern:

- `ghcr.io/<org>/workflow-<workflow_slug>@sha256:<digest>`

## Execution Model

- Outside sandbox:
  - CommandGrid orchestration
  - secret/session setup
  - GhostProxy
- Inside sandbox:
  - workflow executable from FlowSpec image
  - calls to MCP tools supplied from the separate tool repo
  - LLM calls made by workflow code and routed via proxy

## Operational Rules

- pin versions or digests for production jobs
- maintain customer allowlists for workflow selection
- enforce runtime limits (cpu, memory, timeout, token budget)
- record workflow slug/version/digest for audit and billing
