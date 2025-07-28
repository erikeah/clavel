# Clavel

### Pull-based and efficient deployments in nix ecosystem

Source control, which defines the desired infrastructure state. This could be any resource which could be nix evaluated by Crossplane and then applied to nodes.

Crossplane is in charge to evaluate nix code, provide infrastructure status and necessary information to perform deployments from nodes.

A Node could be any, nixos machine, VM or a single user home. The node responsibility is to ask crossplane for deployment updates and report current status of deployment. In fact, crossplane is a node too, and it can managed by itself.

`[Source control] <= [Crossplane] <= [Node]`
