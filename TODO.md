# TODO

## Field masking

Field masking currently only distinguishes empty vs. non-existence for `metadata.finalizers`
(see `pkg/api/core/v1/metadata_converter.go`).

Enhancement (deferred): generalize mask semantics so any masked field can express
"explicitly set to empty" vs. "not part of this update" independently of the field's zero
value, and apply the mask consistently to `spec`, `metadata`, and `status`.

## Deletion / finalizers

Deletion is wired end to end: `genericstore.Watch` emits put/delete/sync events from one
gap-free etcd read, the informer evicts on delete and replaces its cache on the sync marker,
and reconcile finalizes a terminating resource by dropping this controller's finalizer and
deleting once no finalizer blocks it (`internal/controller/evaluation/controller.go`).

Enhancement (deferred):

- the evaluation controller does not *register* `controllerFinalizer`, so no `Evaluation`
  carries one and the clear step only runs when another actor registered it; register it as
  soon as there is cleanup to perform on deletion (the `nix copy` cache removal of an
  Artifact, the owned resources of a `ClavelConfiguration`).

## Evaluation attrs filter

Allow to choose which attr are saved to status.result, this will make evaluation faster and also allow to hide sensible values.

## Result size cap

`status.result` holds the full JSON evaluation result inline in the etcd object, which is
subject to etcd's default ~1.5 MiB request limit. The result is base64-encoded, inflating the
stored payload by roughly 33% over the raw JSON, so a large result makes `Update` fail and the
controller retries forever.

Enhancement (deferred): cap `result` (e.g. 1 MiB) with a `message` noting truncation, or move
results to a separate key referenced by the status.

## Nix definition to RPC

A Nix definition is pure and cannot call the API: an imperative driver must eval it, diff the
result against stored resources, then issue Create/Update/Delete. Gaps:

- no apply driver exists yet; `clavelcontroller` only consumes `Evaluation`s, never produces them
- the definition output needs a versioned schema backed by a proto message; only `Evaluation`
  exists (`api/clavel/core/v1/`), there is no bundle/configuration message
- no atomic or batch apply: RPCs are independent (`internal/core/evaluation_service.go`), so a
  failure mid-apply leaves a partial set; apply must be idempotent and retried
- two eval paths must agree: the driver evals the definition while the controller evals each
  `spec.reference` (`internal/nix/eval.go`); divergent source URLs, lockfiles or ref
  canonicalization make them evaluate different revisions
- `ValidateReference` (`internal/core/evaluation.go`) only rejects spaces/tabs; flake refs need
  canonicalization before diffing or every apply reports spurious churn
- identity is the Nix attr name: renaming a `nixosUnits.<x>` becomes delete + create, because
  `MergeEvaluation` forbids renaming

## Dependency orchestration

- dependencies are invisible to the driver unless Nix emits explicit edges; `nix eval` flattens
  implicit ones (a unit reading another unit's result), and that result only exists in `status`
  after reconcile, so Nix cannot see it during the definition eval
- lazy Nix evaluation tolerates cycles, an apply graph cannot: needs cycle detection plus
  topological order on apply and reverse order on teardown
- reconcile has no ordering (`internal/controller/evaluation/controller.go`): it short-circuits on
  `Succeeded && observedGeneration == generation` and never checks upstream readiness
- no reverse-dependency index: `Informer.Add` enqueues only the changed key, and nothing maps
  an upstream status change to the resources that depend on it — `Metadata.ownerReferences`
  (`api/clavel/core/v1/metadata.proto`) are stored but never read back
- failure semantics undefined: a failed dependency should block its dependents instead of
  retrying them forever through the workqueue

## Nix as source of truth / pruning

- Nix cannot signal deletion: the driver must diff desired vs. actual, which requires ownership
  labels/annotations (`managed-by`, `owner`) — `Metadata`
  (`api/clavel/core/v1/metadata.proto`) can carry them, but nothing writes them yet
- pruning is destructive without a safety net (allowlist, plan/dry-run, `prevent_destroy`
  guard), otherwise a typo in the definition removes live resources
- ownership-scoped listing is not expressible: `EvaluationServiceListRequest` has no selector
- an API-side `Delete` is resurrected by the next apply while a Nix-side removal lingers until
  prune; one side must be declared authoritative (GitOps model) and drift decided
- every pruned resource is deleted through reconcile once it terminates
  (see `Deletion / finalizers`), so the remaining gap is ownership, not deletion

## Concurrent updates

`genericstore.Update` performs a blind `Put` while `MergeMetadata` compares `resourceVersion` in
Go, leaving a TOCTOU window between `FindOne` and `Put`. Needs an etcd transaction comparing
`ModRevision` (compare-and-swap), as `Create` already does.

## Evaluation sandboxing

- `nix eval` on untrusted references can trigger import-from-derivation, network fetches,
  `builtins.getEnv` and `builtins.currentTime`: needs `--restrict-eval`/sandboxing and a network
  policy, and impure inputs break reproducibility
- secrets can leak into `status.result` (see `Evaluation attrs filter` above)

## Nix module surface

- `flake.nix` already references `./flake-module.nix` and `./lib.nix`, and the example imports
  `inputs.clavel.clavelModules.nixos`; none of those files exist yet
- the example hardcodes `github:erikeah/clavel?dir=example#server` while `clavel.url = "./.."`;
  generated unit references must match the real source URL or eval fetches another revision
- module merge semantics for `imports` and list options, option type validation and a definition
  format version are undefined

## Observability

- no dry-run/plan/diff output, no conditions or events beyond `status.phase`, no audit of which
  definition applied a resource