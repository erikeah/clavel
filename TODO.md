# TODO

## Field masking

Field masking currently only distinguishes empty vs. non-existence for `metadata.finalizers`
(see `pkg/api/core/v1/metadata_converter.go`).

Enhancement (deferred): generalize mask semantics so any masked field can express
"explicitly set to empty" vs. "not part of this update" independently of the field's zero
value, and apply the mask consistently to `spec`, `metadata`, and `status`.

## Deletion / finalizers

`internal/genericstore`'s Watch only forwards `EventTypePut`, so deletes never reach the
controller: `EvaluationService.Delete` with finalizers sets a `deletionTimestamp`, but the
reconcile loop only sees the resource disappear from a `List`-less stream.

Enhancement (deferred):

- emit delete events from the store's Watch (requires a `deleted` marker on
  `EvaluationServiceWatchResponse` in `api/clavel/core/v1/evaluation.proto` plus
  `buf generate`),
- let the informer evict on delete and enqueue the key,
- let reconcile clear finalizers before the resource is removed (currently reconcile
  returns nil for any missing key or `deletionTimestamp`).

## Evaluation attrs filter

Allow to choose which attr are saved to status.result, this will make evaluation faster and also allow to hide sensible values.

## Result size cap

`status.result` holds the full JSON evaluation result inline in the etcd object, which is
subject to etcd's default ~1.5 MiB request limit. A large result makes `Update` fail, and the
controller retries forever.

Enhancement (deferred): cap `result` (e.g. 1 MiB) with a `message` noting truncation, or move
results to a separate key referenced by the status.