# Clavel

## Summary

A Nix deployment coordinator designed for simplicity and strong domain alignment, making it highly extensible to new platforms and system types.

Its core function is to evaluate Nix flake sources into structured deployment descriptions, without building software.
Then Agents (remote daemons) poll Claveld for deployment specifications and apply them to target systems (Units).
The system is designed to be **declarative, reproducible**, and **extensible** to many environments (e.g., NixOS, containers, Darwin, routers, IoT).

---

## Ubiquitous Language

| Term              | Type                    | Description                                                                                                                                             |
| ----------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Claveld**       | Service / Server Daemon | A crossplane-like coordinator that evaluates Nix flake sources into structured **Deployments**. It does **not** build software.                         |
| **Source**        | Path                    | A path reference to a Nix flake.                                                                                                                        |
| **Deployment**    | Entity                  | A named, declarative unit of configuration derived from a flake, with a **Specification** describing how to deploy it. Agents know deployments by name. |
| **Agent**         | Actor / Client Daemon   | A worker process responsible for polling Claveld and applying **Deployments** to a target **Unit**.                                                     |
| **Unit**          | Entity / Target         | A machine or environment where an Agent runs and applies deployments — e.g., a Linux system, VM, container, Darwin, router, IoT device, etc.            |
| **Specification** | Object                  | Encapsulates attributes related to direct parent.
| **Metadata**      | Key-Value               | Date of creation, source, misc., etc.

---

## Runtime Use Case Flow

1. A **Source** with a new `nixosConfiguration` is accesible by **Claveld**.
2. **Claveld** performs evaluation and extracts **Deployments** from **Source**.
3. Each **Deployment** has a name and a **Specification** (closures, narinfo, etc.).
4. An **Agent** on a **Unit** polls Claveld and requests a **Deployment** by name.
5. **Claveld** returns the **Deployment.Specification**, allowing the agent to apply it.

---

## Domain Model Sketch

### `Source` 
path to nix sources

---

### `Deployment`
- `name`: string
- `metadata`: { source: Path, sourceHash: <sha256>, ... }
- `specification`: DeploymentSpecification

---

### `DeploymentSpecification`
- `unit`: reference string to Unit
- `template`: NarTemplateSpecification

---

### `NarTemplateSpecification`
- `metadata`: {} // Acts as placeholder
- `narId`: string identifier for NAR (Nix ARchive)
- `substituters`: list of substituters where this nar could be found
- `substituters-public-keys`: list of public keys related to substituters

---

### `Agent`
- `name`: string
- `metadata`: {} // Acts as placeholder for any need
- `specification`: AgentSpecification

---

### `AgentSpecification`
- `deployments`: list of deployments names the agent can request aka DeploymentRef

---

### `Unit`
- `name`: string
- `metadata`: {} // Acts as placeholder for any need
- `specification`: UnitSpecification

---

### `UnitSpecification`
- `type`: e.g., `nixos`, `darwin`, `container`, `iot`, etc.
- `arch`: system architecture (`x86_64-linux`, etc.)

---

## ✅ Domain Invariants

- Claveld never builds software (only evaluates).
- Deployments are uniquely named and immutable once evaluated.
- Specifications must be complete and declarative.
- Units may be heterogeneous (physical, virtual, OS types).
