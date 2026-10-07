# Clavel design

Status: proposal (early-stage WIP).

This document captures the architecture for building resources on top of
`Evaluation`s — the `Nix definition → RPC` pipeline, dependency orchestration, and
deletion as the Nix module is the source of truth. The individual gaps live in
`TODO.md`; this doc is the reasoning behind how they get closed.

## Principles

- **Clavel is Kubernetes, but the artifact is a Nix evaluation result instead of a
  container image.** The API conventions k8s already solved — identity, ownership,
  pruning, finalizers, conditions — are adopted wholesale instead of reinvented.
- **`Evaluation` is the generic "compute a value" primitive.** Every other resource
  is a *consumer* of an `Evaluation`'s `status.result`. Dependencies are therefore
  always "resource X needs the result of evaluation Y", never "X needs Y's Nix
  expression".
- **Data flows through etcd, never through re-evaluation.** A store path is computed
  once by the controller and then `nix copy`'d. This is what makes "no build at
  deploy time" real, and it removes the need for the hard two-phase-eval problem of
  feeding results back into Nix.
- **The definition eval and the per-unit eval are the same primitive** (`nix eval
  --eval-cache --json`) with different references. There is one evaluation mechanism,
  not two eval paths.

## Core model

Every resource message carries `TypeMeta` plus a k8s-style `ObjectMeta`:

```nix
# TypeMeta
apiVersion  # clavel.core/v1
kind        # Evaluation | Artifact | ClavelConfiguration | ...

# ObjectMeta
name                 # immutable; idempotence and configuration key
uid                  # server-assigned, stable for the object's lifetime
resourceVersion      # etcd ModRevision; optimistic concurrency
generation           # bumped on spec change
creationTimestamp
deletionTimestamp
labels               # selectors: owner scoping, dependency queries
annotations
ownerReferences      # ownership: prune, reverse-dependency index, teardown order
finalizers           # block removal until external cleanup runs
```

`Metadata` today (`api/clavel/core/v1/metadata.proto`) already holds `generation`,
`resource_version`, timestamps and `finalizers`; it gains `uid`, `labels`,
`annotations` and `owner_references`.

### Identity and renames

- `name` is the idempotence key and cannot be updated (`MergeEvaluation`,
  `internal/core/evaluation.go`) — matching k8s semantics.
- `uid` is the runtime identity references are written against.
- Renaming a Nix attr is therefore delete + create. The manifest may set an explicit
  `id`/annotation when a stable key across renames is wanted; otherwise attr name
  remains the key.

### Evaluation

Unchanged: `spec.reference` (`<source>#<target>`), `status.{phase, result,
observedGeneration, message}`. `status.result` is the base64-encoded JSON of
`nix eval` — the value every other resource consumes.

## Resource kinds

### Evaluation

The primitive. The controller hands `spec.reference` to `nix eval` and writes the
outcome to `status`.

### Artifact

Lets a consumer pull an already-evaluated store path without building anything:

```proto
message ArtifactSpecification {
  string store = 1;            // https://cache.nixos.org
  ArtifactStorePath store_path = 2;
}
message ArtifactStorePath {
  string eval_ref = 1;         // name of the Evaluation owning the store path
}
```

Reconcile: wait until `eval_ref` is `SUCCEEDED`, read its `status.result`, then
`nix copy --from {store} {result}`. Finalizer: remove the copied path from the local
cache on deletion.

### ClavelConfiguration (the root)

The persistent, first-class record of an applied Nix definition — the thing
`clavelctl apply` creates. It *is* the root resource:

```proto
message ClavelConfigurationSpecification {
  string reference = 1;        // <source>#<evaluation-target>, e.g. .#clavelConfiguration.my-config
}
```

- `spec.reference` points at the definition in the flake.
- `status.result` is the **manifest**: the evaluated list of typed resources
  (`apiVersion`/`kind` + `ObjectMeta` + `spec`), produced by `nix eval --json`.
- `ownerReferences` on every materialized resource point at it, so it owns its
  children.

Because the definition is evaluated into `status.result`, the root records the desired
state in etcd — there is no separate state file. The manifest *is* the state.

A `ClavelConfiguration` whose result is a shell executable (instead of a JSON
manifest) is applied by executing it; the `kind`/content-type discriminator on the
result decides which consumer path runs.

## Apply flow

```
Nix flake (source of truth)
   │  nix eval --json <ref>          sandboxed: --restrict-eval, no IFD, locked flake.lock
   ▼
manifest = list of typed resources
   │  clavelctl apply [--dry-run]    diff vs etcd, then Create/Update/Delete via the API
   ▼
clavelapi (etcd)  ──watch──►  controllers
   │                              ├─ evaluation:       nix eval spec.reference  → status.result
   │                              ├─ artifact:         wait eval_ref SUCCEEDED  → nix copy
   │                              └─ clavelConfiguration: wait eval_ref SUCCEEDED → apply result
   ▼
ownerReferences + finalizers → prune, teardown order, cache cleanup
```

- `clavelctl` is a thin client over the same Connect/gRPC API integrations use —
  the kubectl model. The API is the single write path and the single contract.
- `--dry-run` prints the diff of manifest vs. etcd state (terraform plan /
  kubectl diff); it is a pure function of the two documents.
- Apply is idempotent and retried rather than transactional: a failure mid-apply
  leaves a partial set that the next reconcile converges.
- The watch stream is the controller's only view of etcd. A connection replays the
  current state and terminates it with a sync marker, then streams put/delete events
  for changes; it resumes at the revision the snapshot was read from, so nothing is
  read twice and nothing slips between the two. The informer replaces its cache on
  the sync marker, which is how a reconnect forgets what was deleted while it was
  down.

## Dependency resolution

The manifest encodes the DAG for free: `artifact.spec.storePath.evalRef` and
`clavelConfiguration.spec.reference` *are* the edges.

- **Creation order** — topological sort of the manifest, computed by the driver.
  Cycles are rejected at apply time (Nix's lazy evaluation tolerates them, an apply
  graph cannot).
- **Runtime propagation** — readiness-gating, not scheduling. A consumer's reconcile
  checks its `eval_ref` is `SUCCEEDED` before acting; otherwise it stays `PENDING` and
  is re-enqueued when the upstream status changes. The reverse index comes from
  `ownerReferences` (and a `dependsOn` label when ownership is not the relationship).
- **Failure semantics** — a failed dependency blocks its dependents with a message in
  `status.message`; dependents are not retried independently of their upstream.
- **Teardown** — reverse topological order, driven by the same edges.

No general DAG scheduler lives in Go: Nix resolves definition-level references during
the single definition eval, and readiness-gating resolves runtime references.

## Ownership and pruning

Nix cannot signal deletion — an attr removed from the flake produces no event. The
driver spots it by diffing:

```
prune = { r : r is owned by <root>, r.name ∉ manifest }
```

- Ownership = `ownerReferences` written at apply time, so pruning only ever touches
  resources this root created. Unowned resources are never touched.
- The reverse of the same diff produces creates/updates.
- An API-side `Delete` of an owned resource is re-created by the next apply; the Nix
  definition is authoritative for owned resources (GitOps), while integrations keep
  API access for unowned resources and for `status` writes.
- Deletion is plumbed through the watch stream (`EventType`: put, delete, sync): the
  store reads a snapshot and resumes at that revision, the informer replaces its cache
  on sync and evicts on delete, and reconcile finalizes a terminating resource by
  dropping its finalizer and deleting once none blocks it. What is still missing is a
  finalizer that is actually registered (`TODO.md`).

## Concurrency

Updates are optimistic-concurrency with Kubernetes semantics: a caller must send the
`resourceVersion` it read, and the store enforces the compare atomically rather than in Go.

- `MergeMetadata` rejects a stale `resourceVersion` up front with `Conflict`, so the caller
  re-reads instead of being handed a bad request.
- `genericstore.Update` is an etcd transaction comparing `ModRevision` against that
  revision — the same shape `Create` already used to guard against double-creation. A failed
  precondition returns `DoesNotExist` if the resource is gone and `Conflict` otherwise, and a
  successful write stamps the new revision back onto the object.
- `Conflict` maps to `CodeFailedPrecondition`, which reconcile treats as a retryable error:
  the loser re-reads, sees the object already converged, and stops.

What this buys and what it does not: two controllers racing on the same resource can never
lose a write, but both still *run* the evaluation before finding out who won. Avoiding the
duplicate work needs leader election, which is deferred (`TODO.md`, `Controller HA`).

## Sandbox and reproducibility

The "no build at deploy time" rule doubles as the security boundary:

- `--no-allow-import-from-derivation` — forbids IFD, which is exactly "no building".
- `--restrict-eval` with an `--allowed-uris` allowlist — evaluation cannot reach the
  network or arbitrary store paths; flake inputs arrive earlier via `flake.lock`.
- impurity (`builtins.getEnv`, `builtins.currentTime`) is rejected so evaluations are
  reproducible.
- secrets never enter `status.result` unfiltered — `Evaluation attrs filter`
  (`TODO.md`) is the enforcement point.

## Caveats → solutions

| Caveat (`TODO.md`) | Resolution |
| --- | --- |
| Nix definition → RPC | manifest from `nix eval`; `clavelctl` diffs and applies via the API; root resource records desired state |
| No stable schema | `TypeMeta` (`apiVersion`/`kind`) on every manifest entry; new kinds extend the schema |
| No atomic apply | idempotent, retried convergence instead of transactions |
| Two eval paths | one primitive: `nix eval --json` for both definition and unit references |
| Reference canonicalization | driver normalizes flake refs before diffing; `ValidateReference` tightened |
| Rename = delete+create | k8s semantics: immutable `name`, `uid` for runtime identity |
| Dependency invisibility | manifest carries the edges (`evalRef`); readiness-gating resolves runtime deps |
| Cycles | rejected at apply time |
| No ordering / reverse index | topo-sort for creation; `ownerReferences` reverse index for propagation |
| Never-ending retries | dependents blocked with a `status.message`, not independently retried |
| Deletion not observed | delete events + informer eviction + finalizer clearing |
| Destructive prune | `ownerReferences` scoping: only resources this root created are prunable; `--dry-run` first |
| No selector on List | labels + selector on `ListRequest` |
| Drift authority | root owns its children; definition authoritative for owned, API open for unowned |
| TOCTOU update | CAS transaction on `ModRevision` |
| Untrusted eval | IFD forbidden, `--restrict-eval`, locked inputs |

## Phased implementation

1. **ObjectMeta + TypeMeta** — extend `metadata.proto` (`uid`, `labels`,
   `annotations`, `owner_references`), add `apiVersion`/`kind`; `buf generate`.
   Foundational for everything below. *(done)*
2. **Delete / finalizer plumbing** — the `Deletion / finalizers` item in `TODO.md`:
   typed watch events (put/delete/sync) on `EvaluationServiceWatchResponse`, informer
   eviction and snapshot replace, finalizer clearing plus delete in reconcile.
   *(done, registering the finalizer still pending)*
3. **CAS update** — `genericstore.Update` as a `ModRevision` transaction. *(done)*
4. **Readiness-gating + reverse index** — consumers wait on `eval_ref`; dependents
   re-enqueue on upstream status change; `status.message` carries blocked reasons.
5. **Artifact kind** — proto + converter/setter + reconcile running `nix copy`.
6. **ClavelConfiguration root + clavelctl apply** — proto, manifest evaluation,
   diff/apply via the API, `ownerReferences`, prune, `--dry-run`.
7. **Sandboxing** — eval flags (`--restrict-eval`, `--no-allow-import-from-derivation`)
   in `internal/nix`.
8. **Observability** — conditions/events beyond `status.phase`, plan output, audit of
   which root applied a resource.

Deferred items already tracked in `TODO.md` (field masking, attrs filter, result size
cap) are independent and slot in wherever convenient.
