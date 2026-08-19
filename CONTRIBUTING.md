# How to Contribute

The Operator Certification Operator is part of the [Red Hat Operator Ecosystem][operator_ecosystem_org]. Contributions are accepted via GitHub pull requests. This document outlines conventions on commit message formatting, development workflow, and other resources to help get contributions into this project.

## Getting Started

The Operator Certification Operator provisions resources for the operator certification pipeline in Kubernetes clusters. It's designed for Red Hat partners certifying their operators on their own infrastructure and supports multi-tenant scenarios across all namespaces.

- Fork the repository on GitHub.
- Install the CLI tools listed under [Dev Setup][dev_setup_docs] and confirm
  they're on your `PATH`.
- Follow the [Testing Locally][testing_locally] section to run the operator in your development environment.

## High-Level Codebase Map

- **Entry point**: `cmd/main.go` initializes the controller manager, registers
  the scheme with required API types (Tekton, OpenShift Image/Security, OLM),
  and starts the `OperatorPipelineReconciler`.
- **API types**: `api/v1alpha1/operatorpipeline_types.go` defines the
  `OperatorPipeline` Custom Resource with its spec (image references, secrets,
  repository config) and status (conditions for each reconciled resource).
- **Controller**: `internal/controller/operatorpipeline_controller.go`
  implements the main reconciliation loop — watches `OperatorPipeline` resources,
  handles finalizers, and delegates to modular reconcilers for each managed
  resource type.
- **Reconcilers**: `internal/reconcilers/` contains individual reconcilers that
  implement the `Reconciler` interface (`Reconcile(ctx, pipeline) (bool, error)`):
  - `certified_image_stream.go` — manages the certified operator ImageStream
  - `marketplace_image_stream.go` — manages the marketplace ImageStream
  - `pipeline_dependencies.go` — creates Tekton Pipelines, Tasks, ServiceAccounts,
    Roles/RoleBindings, and SecurityContextConstraints
  - `pipeline_git_repo.go` — sets up the Git repository Secret for the pipeline
  - `status.go` — updates the `OperatorPipeline` status conditions
- **Objects**: `internal/objects/objects.go` provides helpers to construct
  Kubernetes objects with proper ownership references and labels.
- **Pyxis integration**: `internal/pyxis/` handles communication with Red Hat's
  Pyxis service for certification submission and validation.
- **Config & deployment**: `config/` contains Kustomize manifests for CRDs
  (`config/crd`), RBAC (`config/rbac`), the manager deployment (`config/manager`),
  and sample CRs (`config/samples`).

## Building and Testing Your Change

```bash
make build         # build the manager binary
make generate      # generate code and verify no uncommitted changes
make manifests     # generate manifests and verify no uncommitted changes
make tidy          # tidy go modules and verify no uncommitted changes
make vet           # run go vet
make fmt           # format code
make lint          # run linter
make test          # run tests
```

These steps match the CI validation workflow and must all pass before a PR is accepted.

Run your change end-to-end in a real cluster before opening a PR. The operator reconciles `OperatorPipeline` custom resources — see [installation documentation][install_docs] for how to create and validate CRs.

## Reporting Bugs and Creating Issues

Reporting bugs is one of the best ways to contribute. Before creating a bug report, please check that an issue reporting the same problem does not already exist.

To make the bug report accurate and easy to understand, please create bug reports that are:

- **Specific**: Include as many details as possible — which version, what environment, what configuration
- **Reproducible**: Include the steps to reproduce the problem
- **Isolated**: Try to isolate and reproduce the bug with minimum dependencies
- **Unique**: Do not duplicate existing bug reports
- **Scoped**: One bug per report

If any part of the project has bugs or documentation mistakes, please open an issue. We treat bugs and mistakes seriously and believe no issue is too small.

## Contribution Flow

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from `main`
- Make commits of logical units
- Make sure commit messages are in the proper format (see below)
- Push changes in a topic branch to a personal fork of the repository
- Submit a pull request to this repository
- The PR must receive approval from maintainers

Thanks for contributing!

### Code Style

The coding style suggested by the Go community is used in this project. See the [Go Code Review Comments][golang_style_doc] for details.

Please follow this style to make the codebase easy to review, maintain, and develop.

### Commit Message Format

We follow a convention for commit messages that answers two questions: what changed and why. The subject line should feature the what and the body of the commit should describe the why.

```text
cmd: add the certify sub-command

this adds the certify sub-command to submit test results to Red Hat for certification.

Fixes #61
```

The format can be described more formally as follows:

```text
<subsystem>: <what changed>
<BLANK LINE>
<why this change was made>
<BLANK LINE>
<footer>
```

The first line is the subject and should be no longer than 70 characters, the second line is always blank, and other lines should be wrapped at 80 characters. This allows the message to be easier to read on GitHub as well as in various git tools.

[operator_ecosystem_org]: https://github.com/redhat-openshift-ecosystem
[dev_setup_docs]: https://github.com/redhat-openshift-ecosystem/operator-certification-operator/tree/main/docs/dev/README.md#dev-setup
[install_docs]: https://github.com/redhat-openshift-ecosystem/operator-certification-operator/tree/main/docs/INSTALLATION.md
[golang_style_doc]: https://github.com/golang/go/wiki/CodeReviewComments
[testing_locally]: https://github.com/redhat-openshift-ecosystem/operator-certification-operator/tree/main/docs/dev/README.md#testing-locally
