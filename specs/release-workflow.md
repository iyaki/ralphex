# Release Workflow

Status: Implemented

## Overview

### Purpose

- Define the GitHub Actions workflow used to build and publish Ralph release artifacts.
- Standardize release triggers, artifact naming, and checksum generation.

### Goals

- Publish release binaries for supported platforms from semantic version tags.
- Attach all binaries and a checksum manifest to the GitHub Release.
- Support both automatic tag-based releases and manual release runs.

### Non-Goals

- Package manager publishing (Homebrew, apt, scoop).
- Code signing and notarization.
- Container image publishing.
- Detached signatures (.sig files) for release artifacts.

### Scope

- In scope: workflow triggers, build matrix, artifact upload, checksum generation, and release publication.
- Out of scope: changelog generation policy and external distribution channels.

## Architecture

### Workflow file

```
.github/workflows/release.yml
```

### Component diagram (ASCII)

```
+------------------------------+
| Trigger (tag or manual run)  |
+--------------+---------------+
               |
               v
+--------------+---------------+
| Build matrix (GOOS/GOARCH)   |
+--------------+---------------+
               |
               v
+--------------+---------------+
| Artifact collection + SHA256 |
+--------------+---------------+
               |
               v
+--------------+---------------+
| GitHub Release publication   |
+------------------------------+
```

### Data flow summary

1. Workflow starts on `push` to `v*` tags or manual `workflow_dispatch` with a `tag` input.
2. Matrix builds compile `./cmd/ralph` for each target platform.
3. Build artifacts are uploaded and then collected in a publish job.
4. Publish job generates `checksums.txt` from all release binaries.
5. Workflow creates or updates a GitHub Release and uploads binaries plus checksums.

## Data model

### Core entities

- ReleaseTag
  - Format: semantic version style `v*` (for example `v1.0.0`).
  - Source: `github.ref_name` for tag pushes or `workflow_dispatch` input.

- ReleaseArtifact
  - Naming: `ralph_<tag>_<goos>_<goarch>[.exe]`.
  - Binary source: `go build ./cmd/ralph`.

- ChecksumManifest
  - File: `checksums.txt`.
  - Algorithm: SHA-256.

### Relationships

- One `ReleaseTag` maps to multiple `ReleaseArtifact` files.
- One `ChecksumManifest` covers all artifacts produced for that tag.

### Persistence notes

- Binaries and `checksums.txt` are persisted as GitHub Release assets.
- Temporary build artifacts exist only during workflow execution.

## Workflows

### Automatic release (tag push)

1. Push a tag matching `v*`.
2. Workflow builds binaries for all configured targets.
3. Workflow publishes/updates the matching GitHub Release with artifacts.

### Manual release (workflow_dispatch)

1. Trigger `Release` workflow from GitHub Actions UI.
2. Provide the `tag` input and optionally keep `create_tag=true` to create the tag if it does not already exist.
3. Workflow checks out that tag, builds artifacts, and publishes release assets.

## APIs

- GitHub Actions events: `push`, `workflow_dispatch`.
- GitHub release API usage is mediated by `softprops/action-gh-release`.

## Configuration

- Workflow trigger patterns:
  - `push.tags: ["v*"]`
  - `workflow_dispatch.inputs.tag` (required).
  - `workflow_dispatch.inputs.create_tag` (optional boolean, defaults to `true`).
- Build environment:
  - `CGO_ENABLED=0`
  - `GOOS` / `GOARCH` from matrix.

## Permissions

- Requires `contents: write` to create/update GitHub Releases and upload assets.

## Security considerations

- Release assets are built in CI from tagged refs; tags should be protected.
- Checksums support end-user integrity verification.
- Secrets are not required beyond GitHub-provided token permissions.

## Dependencies

| Dependency                    | Purpose                         |
| ----------------------------- | ------------------------------- |
| `actions/checkout`            | Check out source at release ref |
| `actions/setup-go`            | Configure Go toolchain          |
| `actions/upload-artifact`     | Persist matrix build outputs    |
| `actions/download-artifact`   | Collect outputs for publishing  |
| `softprops/action-gh-release` | Create/update GitHub Release    |

## Verifications

- Pushing a tag like `v0.1.0` triggers `Release` workflow.
- Workflow uploads one binary per matrix target.
- `checksums.txt` is generated and uploaded to the release.
- Downloaded artifact checksums match entries in `checksums.txt`.
