# AGENTS.md

Clavel evaluates Nix flake references — `evaluation`s — into deployment descriptions (early-stage WIP). A flake reference already encodes source and target in one string (`<source>#<evaluation-target>`); `Evaluation` is the core entity, stored via the API. Two runtime binaries: `clavelapi` (Connect/gRPC-over-h2c API backed by etcd) and `clavelcontroller` (watches evaluations via the API and reconciles them). `nix` is a **runtime** dependency, not just a build tool — evaluation happens via `nix eval --eval-cache --json`.

## Commands

Dev environment comes from the Nix flake (`direnv`: `.envrc` → `use flake .`). Tools and helper scripts (`develop-*`) are only on PATH inside `nix develop`; they are defined in `flake.nix`.

```sh
go build ./...        # builds; clavelapi, clavelcontroller, nix-parallel-eval
go test ./...         # no tests exist yet; used for compile-check
buf generate          # regenerate committed protobuf code (see below)
buf lint && buf breaking  # proto checks; buf.yaml points at api/
nixfmt-tree           # nix formatter (flake formatter); gofmt for Go
```

Dev/watch helpers (run from a `nix develop` shell):
- `develop-start-services` — starts etcd, required by `clavelapi` (hardcoded `localhost:2379`).
- `develop-watch-clavelapi` — hot-reloads clavelapi on `PORT=8080`.
- `develop-watch-clavelcontroller` — hot-reloads the controller (hardcoded `http://localhost:8080`).
- `develop-debug-clavelapi` — runs the API under delve.
- `go run ./cmd/nix-parallel-eval <ref>#<target> [<ref>#<target> ...]` — standalone debug CLI evaluating one or more flake references in parallel.

Integration flow: start etcd → `clavelapi` (PORT 8080) → `clavelcontroller`.

## Protobuf codegen

- Protos live under `api/clavel/**`. Generated Go (`*.pb.go`, connect stubs) is **committed** under `pkg/api/` and generated with `buf generate` (see `buf.yaml` + `buf.gen.yaml`; `out: .` + `module=` opt maps output to `pkg/api/`).
- Always run `buf generate` after editing a `.proto`. Hand-written converter/setter files (e.g. `pkg/api/core/v1/evaluation_converter.go`, `evaluation_setter.go`) live next to generated code and are also tracked — extend them rather than editing `.pb.go` by hand.

## Structure

- `cmd/clavelapi` — API server; `PORT` env (default 80), no TLS; Connect RPC over h2c.
- `cmd/clavelcontroller` — connects to clavelapi's Watch RPC; a Kubernetes-style controller (`internal/controller`: informer cache → rate-limited dedup workqueue → worker pool) reconciles evaluations.
- `internal/genericstore` — generic etcd-backed KV store: resources JSON-serialized under `/<path>/<name>`; `resource_version` = etcd `ModRevision`.
- `internal/controller` — generic K8s-style controller plumbing: `WorkQueue` (rate-limited, deduplicating, retry backoff), `Informer` (watch-backed cache), and `Controller` (worker pool running a `ReconcileFunc` per key).
- `internal/controller/evaluation` — the per-resource `Evaluation` controller: wires the generic plumbing to the API's Watch RPC and runs reconcile (mirrors k8s `pkg/controller/<kind>`).
- `internal/transport` — Connect RPC transport layer: `evaluation` handler and `interceptors` (error interceptor), mirroring k8s' `pkg/registry/<group>/<resource>` + storage stratum.
- `internal/core` — domain types (Metadata, Evaluation) plus the `Evaluation` service/store; `internal/fieldmaskcommander` — field-mask updates.
- `internal/utils/parallel_nix_eval.go` — calls out to `nix eval` for evaluation references (`<source>#<evaluation-target>`).
- `example/` — a consumer flake importing this repo as `path:../`, exercising `flakeModule` + `lib.mkNixosUnit`; useful for sanity-checking flake-level changes.

## Conventions

- Errors wrap the sentinel package `internal/exceptions` via `errors.Join`.
- Commit messages are lowercase, conventional-style (`feat:`, `refactor:`, `chore:`, `format:`).
- `.direnv` and `.develop*` (etcd data dir) are gitignored; never commit them. No CI and no tests exist yet.