# TODO

## Field masking

Field masking currently only distinguishes empty vs. non-existence for `metadata.finalizers`
(see `pkg/api/core/v1/metadata_converter.go`).

Enhancement (deferred): generalize mask semantics so any masked field can express
"explicitly set to empty" vs. "not part of this update" independently of the field's zero
value, and apply the mask consistently to `spec`, `metadata`, and `status`.