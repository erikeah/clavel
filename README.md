# Clavel

A Kubernetes-like control plane for the Nix ecosystem (NixOS, home-manager, Liminix, containers, …).

Clavel turns Nix flake references into deployment descriptions **without building software** — evaluation happens with `nix eval --eval-cache --json`, which yields a derivation hash/store path, not an output. Desired state is declared in a Nix flake, stored through `clavelapi`, and materialized by `clavelcontroller` via per-integration **action modules**.

The system is designed to be **declarative, reproducible**, and **extensible**: new integrations (`NixosUnit`, `ContainerUnit`, `HomeManagerUnit`, …) register a custom resource kind at runtime — schemas, validation, and deploy logic are all data, so third parties can extend Clavel without recompiling the core.

---

## Architecture

```
  consumer flake                         runtime (etcd-backed)
 ┌───────────────────────┐              ┌───────────────────────────┐
 │ clavelDefinition {    │ apply (nix  │ clavelapi          watch  │
 │   imports = [         │ eval + push)│  - CRD registry           │
 │     clavelModules.nixos]; ─────────▶│  - generic resource store │ ──▸ clavelcontroller
 │   nixosUnits."foo"… } │              │  - dynamic spec validation│       (generic reconciler)
 └──────────┬────────────┘              └─────────────┬─────────────┘        │
            │ clavelModules emits                      ▲                     │
            │ CRDs + resources + evaluations           │ status written back  │
            │                                          │                      ▼
            │                                nix eval derivation ◀── action module
            │                                     (evalRef → hash)      (reference + apply)
            └───▶ clavelctl apply <flake>
```

Three binaries:

- **`clavelapi`** — the API server (like `kube-apiserver`). Stores state in etcd, keeps a registry of registered **kinds**, and validates resource specs against each kind's schema at runtime. Connect/gRPC over h2c, no TLS.
- **`clavelcontroller`** — a generic reconciler (like a Kubernetes operator). Watches resources via the API, evaluates the referenced `Evaluation`, writes its status back, and materializes the deployment by invoking the kind's **action module**.
- **`clavelctl`** — the CLI to evaluate a configuration from a flake and push it to `clavelapi`.

---

## Core concepts

### Evaluation

`Evaluation` is the core entity. It pairs a flake `reference` of the form `<source>#<evaluation-target>` with core metadata:

```
name: <name>
metadata: { generation, resourceVersion, creationTimestamp, deletionTimestamp, finalizers }
spec:     { reference: "<source>#<evaluation-target>" }
status:   { phase: Ready|Failed, result: { hash, storePath } }
```

An evaluation only *evaluates* (`nix eval --eval-cache --json`) — it never builds. Evaluations are reusable across kinds; a kind's resource merely references one by name.

### CustomResourceDefinition (CRD)

A `CustomResourceDefinition` registers a new resource kind (group/version/kind/plural) together with:

- `schema` — a serialized `FileDescriptorSet` (the spec message's proto schema, including embedded `buf.validate` constraints).
- `specMessage` — the fully-qualified proto message name of the spec.
- `actionModule` — a flake reference to the Nix module that knows how to turn an instance into a deployment.

CRD schemas are **immutable once registered**: reapplying an identical definition is a no-op, a conflicting re-apply is rejected, and deletion is blocked while instances exist.

### Resource

A `Resource` is an instance of a registered kind. Its `spec` is a protojson message, validated by `clavelapi` at create/update time using `dynamicpb` + `protovalidate` against the kind's schema. Resources reference the `Evaluation` responsible for producing the artifact (e.g. a NixOS toplevel derivation).

### Apply descriptor

The output of `clavelDefinition` — `{ customResourceDefinitions, resources, evaluations }` — is the declarative form of desired state. `clavelctl apply` evaluates it from the flake and pushes each part in dependency order (CRDs first, then evaluations, then resources).

### Action module

The kind's "controller logic", written in Nix. A module exporting two functions, invoked by `clavelcontroller` via `nix eval --json <flake>#<attr> --apply 'f: f …'`:

- `reference evalRef` → the flake reference actually evaluated. This is where integrations *reinterpret* a generic reference for their domain (e.g. `#host` → `#nixosConfigurations.host.config.system.build.toplevel`).
- `apply specJson evalRef derivation` → a deployment action, e.g.:

  ```json
  { "type": "command", "command": ["nix-env", "--profile", "/nix/var/nix/profiles/system", "--install", "/nix/store/….drv"] }
  { "type": "ssh", "host": "worker", "command": ["nixos-rebuild", "switch"] }
  ```

---

## The deployment loop

For each resource of a registered kind, the controller:

1. Decodes the resource spec via the kind's schema and reads the referenced `Evaluation` name (conventionally the `evaluation` string field).
2. Loads the `Evaluation`, invokes `<actionModule>.reference` to transform its reference, and runs `nix eval --json` on the result to obtain the derivation store path.
3. Writes `EvaluationStatus` back (`Ready`/`Failed` + `hash`/`storePath`).
4. Invokes `<actionModule>.apply` to get a deploy action and executes it (`command` runs locally, `ssh <host>` remotely).

Reconciling an already-deployed derivation skips re-deployment.

---

## Defining an integration

An integration is a clavel **module** plus an **action module**, backed by a proto-defined schema. The `nixos` integration is the reference example.

1. **Spec schema** (`api/clavel/units/nixos/v1/nixos_unit.proto`):

   ```proto
   message NixosUnitSpec {
     string evaluation = 1;
     string profile = 2;
     repeated Strategy strategies = 3;
   }
   ```

   After `buf generate`, `clavel-gen` (a protoc plugin registered in `buf.gen.yaml`) derives the Nix descriptor and a typed spec submodule. Regeneration is driven by `buf generate` with per-integration parameters in the plugin config:

   ```sh
   buf generate
   ```

   This writes `nix/generated/nixos.nix` with the base64 `FileDescriptorSet` (the CRD schema) and a Nix `spec` type mirroring the fields.

2. **Clavel module** (`modules/nixos.nix`) registers the CRD, declares the user-facing `<kind>s` option, and emits resources/evaluations into the apply descriptor.

3. **Action module** (`modules/actions/nixos.nix`) implements `reference` + `apply`.

4. **Flake wiring** (`flake.nix`) exposes `clavelModules.nixos` and `clavelActions.nixos`.

Consumers import the module and declare desired state (see [`example/flake.nix`](example/flake.nix)):

```nix
clavelConfigurations.default = inputs.clavel.lib.clavelDefinition {
  imports = [ inputs.clavel.clavelModules.nixos ];
  nixosUnits."atadecer" = {
    name = "atardecer";                                # optional, defaults to the attr name
    evaluation = {
      name = "atardecer";                              # optional, defaults to the resource name
      reference = "github:erikeah/configurations#atardecer";
    };
    profile = "/nix/var/nix/profiles/system";          # optional
    strategies = [ { type = "local"; } ];              # optional
  };
};
```

---

## Trying it out

Requires a Nix installation and Go inside `nix develop`.

```sh
nix develop
develop-start-services          # etcd on localhost:2379
develop-watch-clavelapi         # clavelapi on PORT=8080 (hot-reload)

# in another shell, push the example configuration
go run ./cmd/clavelctl apply "path:./example" --address http://localhost:8080

# watch it get deployed
develop-watch-clavelcontroller
```

The example flake imports clavel as `github:erikeah/clavel`, so it resolves once the project is published; while iterating on an unpublished checkout, point the input at the working tree instead (`clavel.url = "path:../"`).

`clavelctl` flags: `--config <name>` (default `default`, selects `#clavelConfigurations.<name>`) and `--address <host>` (default `http://localhost:8080`).

---

## State & storage

State lives in etcd:

- evaluations: `/evaluations/<name>`
- CRDs: `/crd/<group>/<version>/<kind>`
- resources: `/resource/<group>/<version>/<plural>/<name>`

`resource_version` mirrors etcd's `ModRevision` and drives optimistic-concurrency on updates.

## Development

```sh
go build ./...          # clavelapi, clavelcontroller, clavelctl, clavel-gen, nix-parallel-eval
go test ./...           # compile check + internal/core unit tests
buf generate            # regenerate committed protobuf code (see below)
buf lint && buf breaking
gofmt -l . ; nixfmt-tree
```

- Protos live under `api/clavel/**`; generated Go is **committed** under `pkg/api/`; hand-written converters/setters live next to it (`*_converter.go`, `*_setter.go`).
- Always run `buf generate` after editing a `.proto` (this regenerates the Go bindings and the `nix/generated/*.nix` descriptors via the `clavel-gen` plugin registered in `buf.gen.yaml`).
- The module system for `clavelDefinition` lives in `lib.nix`; per-kind services/stores live in `internal/core/{metadata,evaluation,crd,resource}*.go`; descriptor-based spec validation in `internal/core/spec_validation.go`.

> Work in progress — the architecture above is implemented end-to-end for the `nixos` integration; more integrations and hardening (namespaces, finalizers, constraint generation) are on the roadmap.