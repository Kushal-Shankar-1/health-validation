---
slug: /
---

# NVIDIA Health Validation

NVIDIA Health Validation runs on a GPU node, evaluates hardware and driver state
against calibrated thresholds, and emits a structured JSON verdict. The verdict
answers one question: is this node ready to accept the next workload?

It is node-local and on-demand. Each invocation evaluates one node's
measurements and produces exactly one verdict — there is no state carried
between runs and no cross-node analysis. Run it before dispatching work to a
node; the verdict is a point-in-time snapshot.

- [API Schema](/reference/schema) — the verdict and check-result type
  definitions, including how enums and struct fields serialize in JSON.
