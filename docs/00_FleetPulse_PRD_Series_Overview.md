# FleetPulse PRD Series Overview

## Phased product requirements for the agent, API, deployment, lifecycle, and release system

# Why this is split up
FleetPulse is large enough that one monolithic PRD would be hard to implement, test, and review. This series breaks the work into four buildable steps plus a roadmap overview.

# PRD set
PRD 1 — Core Agent and API — Telemetry collection, normalized schema, deployment targets, and API contract.
PRD 2 — Security and Operations — Authentication, token lifecycle, identity, logging, reliability, and permission model.
PRD 3 — Installation, Uninstallation, and Packaging — systemd, Windows services, macOS launch daemons, Docker persistence, and release packaging.
PRD 4 — Testing, CI/CD, and Release — Automated tests, GitHub Actions pipelines, versioning, artifact integrity, and release gates.

# Suggested execution order
1. Build the core collector and API contract first.
1. Add operational controls and security behavior next.
1. Then ship installers, uninstallers, and packaging for every target.
1. Finally automate testing and release pipelines before broad rollout.

# Design principle
Every later PRD must preserve the same stable API schema and the same deployment targets defined in PRD 1. Later phases may add optional capabilities, but they must not break the earlier contract.
Deliverable note: Each PRD is intentionally usable on its own, but together they define the full FleetPulse product.
