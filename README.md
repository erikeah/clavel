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
```

Evaluations are stored via `clavelapi` (etcd-backed) and watched by `clavelcontroller`, which dispatches rules against them. The reference is handed to `nix eval <reference> --eval-cache --json` at evaluation time.

---

Work in progress! Clavel is in early development stage.