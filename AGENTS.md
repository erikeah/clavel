# AGENTS.md

Clavel evaluates Nix flake sources into deployment descriptions (early-stage WIP). Two runtime binaries: `clavelapi` (Connect/gRPC-over-h2c API backed by etcd) and `clavelcontroller` (watches sources via the API and dispatches with rules). `nix` is a **runtime** dependency, not just a build tool — evaluation happens via `nix eval --eval-cache --json`.

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
- `go run ./cmd/nix-parallel-eval <flake-ref>` — standalone debug CLI for flake eval.

Integration flow: start etcd → `clavelapi` (PORT 8080) → `clavelcontroller`.

## Protobuf codegen

- Protos live under `api/clavel/**`. Generated Go (`*.pb.go`, connect stubs) is **committed** under `pkg/api/` and generated with `buf generate` (see `buf.yaml` + `buf.gen.yaml`; `out: .` + `module=` opt maps output to `pkg/api/`).
- Always run `buf generate` after editing a `.proto`. Hand-written converter/setter files (e.g. `pkg/api/source/v1/source_converter.go`, `source_setter.go`) live next to generated code and are also tracked — extend them rather than editing `.pb.go` by hand.

## Structure

- `cmd/clavelapi` — API server; `PORT` env (default 80), no TLS; Connect RPC over h2c.
- `cmd/clavelcontroller` — connects to clavelapi's Watch RPC and fans events into `internal/genericdispatcher` rules.
- `internal/genericstore` — generic etcd-backed KV store: resources JSON-serialized under `/<path>/<name>`; `resource_version` = etcd `ModRevision`.
- `internal/core` — domain types; `internal/source/api` — service/store over the generic store; `internal/fieldmaskcommander` — field-mask updates.
- `internal/utils/parallel_nix_eval.go` — calls out to `nix eval` for source evaluation.
- `example/` — a consumer flake importing this repo as `path:../`, exercising `flakeModule` + `lib.mkNixosUnit`; useful for sanity-checking flake-level changes.

## Conventions

- Errors wrap the sentinel package `internal/exceptions` via `errors.Join`.
- Commit messages are lowercase, conventional-style (`feat:`, `refactor:`, `chore:`, `format:`).
- `.direnv` and `.develop*` (etcd data dir) are gitignored; never commit them. No CI and no tests exist yet.