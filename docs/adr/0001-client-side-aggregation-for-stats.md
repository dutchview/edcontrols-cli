# Stats commands aggregate client-side

The EdControls API has no aggregation/group-by endpoint, so `tickets stats` and `audits stats` fetch all matching documents (via the search endpoint with `includeFields` trimmed to the grouping fields) and aggregate in the CLI. The goal of the feature is to keep raw tickets out of the LLM context of chat-tool integrations, not to reduce wire traffic — client-side aggregation achieves that fully. The command's interface deliberately looks like a server-side stats API so the internals can be swapped if the backend ever grows aggregations.

## Consequences

- Result sets are hard-capped at 50,000 documents; beyond that the command aborts with a hint to narrow the filters. It never silently truncates — a wrong count is worse than no count.
- Per-invocation latency scales with result-set size (one paged request per ~500 documents).
