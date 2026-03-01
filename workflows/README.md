# Workflows

Each child directory under `workflows/` is an independently versioned workflow component.

Minimum expected files:

- `main.go` (or equivalent entrypoint)
- `go.mod` (or equivalent package manifest)
- `Dockerfile`
- `workflow.yaml`
# Workflows

Each child directory under `workflows/` is an independently versioned workflow component.

Example:

- `workflows/echo`
- `workflows/hello-weather`
- `workflows/gmail-monthly-report`
- `workflows/discord-weekly-digest`

Each workflow directory should include:

- `main.go` (or runtime entrypoint)
- `go.mod` (or language package manifest)
- `Dockerfile`
- `workflow.yaml` (workflow metadata)
