# Clavel

## Summary

Its core function is to evaluate Nix flake **evaluations** into structured deployment descriptions, without building software.
The system is designed to be **declarative, reproducible**, and **extensible** to many environments (e.g., NixOS, containers, Darwin, routers, IoT).

## Evaluations

`Evaluation` is the core entity. It pairs a flake `reference` string of the form `<source>#<evaluation-target>` — a source and a target encoded in one flake reference — with core metadata like generation, resource version, and finalizers.

```
name: <name>
metadata: { creationTimestamp, generation, resourceVersion, finalizers }
spec: { reference: "<source>#<evaluation-target>" }
status: { phase, result, observedGeneration, message }
```

Evaluations are stored via `clavelapi` (etcd-backed) and watched by `clavelcontroller`, which reconciles them. The controller hands `spec.reference` to `nix eval <reference> --eval-cache --json` and writes the outcome back to `status`:

- `phase` — `SUCCEEDED`, `FAILED`, or `PENDING` (`UNSPECIFIED` before the first reconciliation).
- `result` — the raw JSON returned by `nix eval`, base64-encoded so it can be stored and transmitted without escaping. Empty when the evaluation failed.
- `observedGeneration` — the `metadata.generation` this status reflects; changing the spec bumps the generation and re-triggers evaluation.
- `message` — human-readable detail, e.g. the evaluation error when `phase` is `FAILED`.

---

Work in progress! Clavel is in early development stage.