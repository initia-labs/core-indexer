# SQL injection review and regression coverage plan

## Outcome and scope

The source review found no confirmed exploitable SQL injection in the examined
HTTP-to-database and blockchain-to-database paths. This document plans regression
coverage and CodeQL triage; it does not claim a vulnerability fix or a regression.
No production behavior changes are proposed by this planning PR.

The review examined commit `809f285492fe43b15e493c1b27d0f9203a48481b` in
`initia-labs/core-indexer`. The planning PR is based on `d5f8ae4` on `main`, whose
tracked tree is identical to the reviewed commit. The sibling `initia-api`
repository was not reviewed.

Covered inputs include HTTP path/query parameters, sorting, dynamic filters,
pagination, transaction memo, proposal metadata, blockchain event attributes,
NFT metadata, and shared database helpers. Registered API routes were GET-only;
no request-body parsing path to SQL was found. Searches included concatenation,
`fmt.Sprintf`, `strings.Builder`, raw SQL, GORM builders, and execution helpers.

## Evidence and boundaries

| Path | Source to sink and existing protection |
| --- | --- |
| API search and lists | Handler input flows through services to parameterized `Where` predicates, including LIKE/regex/IN values. See `api/repositories/proposal.go:87` and `api/repositories/nft.go:49`. |
| Account filters | `api/repositories/account.go:142-164` interpolates only seven literal map keys; request booleans select predicates and values remain parameters. |
| Validator sort | `api/repositories/validator.go:53-114` selects constant columns/expressions; direction is boolean. Unknown sort values use the existing default. `Raw: true` expressions are constants. |
| Module stats | `api/repositories/module.go:255-262` constructs a data ID, then binds it three times in a constant raw query. |
| Pagination | `api/dto/pagination.go:33-87,113-155` parses integers/booleans and decoded cursors; raw text is not SQL. Large valid offsets are a separate load concern. |
| Memo | `pkg/txparser/tx.go:175` maps memo to a transaction, then `generic-indexer/indexer/block_results.go:95` calls the insert helper ending in `pkg/db/db.go:390` `CreateInBatches`. NUL normalization is not SQL escaping; binding is the protection. |
| Proposal metadata | `informative-indexer/indexer/state-tracker/state_update_manager.go:141-166` maps RPC metadata to fields/JSON; `batch_insert.go:190` calls the helper ending in `pkg/db/db.go:244` `CreateInBatches`. |
| Events and NFT data | `event-indexer/indexer/block_result.go:30-44` stores event keys/values as fields. `pkg/db/db.go:740-772` binds NFT/collection update values. |
| Bulk image update | `pkg/db/db.go:175-191` formats generated placeholder numbers only, with addresses/images passed separately to `Exec`. |
| Pruning tables | `pkg/db/db.go:505-545` validates identifiers against the three literal entries in `pkg/db/valid_tables.go`; thresholds are parameters. |
| Other formatting | `pkg/db/count.go:20` formats a typed integer duration. Migration database identifiers use constant prefixes plus numeric timestamps. |

Resolved source inspected: GORM 1.30.0, PostgreSQL driver 1.6.0. The dialect
creates `$n` placeholders and execution callbacks pass SQL and `Statement.Vars`
separately to `ExecContext`/`QueryContext`. JSON/JSONB types return driver values.

`pkg/db/db.go:939-940` contains an unchecked formatted `TruncateTable` helper,
but repository-wide caller search found no caller. It is an optional hardening
candidate, not a demonstrated reachable vulnerability. If retained for a real
caller, use a dedicated explicit identifier allowlist matching that use case;
do not blindly reuse the pruning allowlist or invent a new runtime caller.

## Delivery plan: small, independently reviewable PRs

Each PR should answer one review question. Start with PRs 1–4, merging the
recorder foundation before dependent tests; do not open all thirteen at once.
These are preventive coverage tasks, not thirteen confirmed vulnerabilities.

Estimates are developer-days for one developer familiar with Go and this repo,
including test implementation and CI fixes but excluding review wait time.
They assume build dependencies are available and no new vulnerability requires
remediation. The finer split totals approximately **8–13 days**; small-PR
coordination adds overhead compared with the original three broad PRs.

| PR | Scope / suggested title | Estimate | Dependency |
| --- | --- | --- | --- |
| 1 | `test: capture SQL arguments for validator search` | 0.5–1 day | None |
| 2 | `test: preserve validator sort allowlist and fallback` | 0.5 day | PR 1 |
| 3 | `test: constrain account dynamic filter SQL` | 0.5 day | PR 1 |
| 4 | `test: bind module IDs in raw stats queries` | 0.5 day | PR 1 |
| 5 | `test: bind proposal and NFT search filters` | 0.5–1 day | PR 1 |
| 6 | `test: reject malformed pagination before repository calls` | 0.5 day | None |
| 7 | `test: bind blockchain memo and preserve normalization` | 0.5–1 day | PR 1 recorder pattern |
| 8 | `test: bind proposal metadata and content JSON` | 0.5–1 day | PR 7 DB test fixture |
| 9 | `test: bind event attributes and NFT metadata` | 0.5–1 day | PR 7 DB test fixture |
| 10 | `test: bind validator image batch updates` | 0.5 day | PR 7 DB test fixture |
| 11 | `test: enforce pruning table allowlist` | 0.5 day | PR 7 DB test fixture |
| 12 | `test: round-trip memo and metadata in PostgreSQL` | 1–2 days | PRs 7–8 |
| 13 | `ci: establish Go CodeQL baseline` | 1–3 days | None; inspect hosted setup first |

Use the same recorder approach across modules, but do not introduce an `api`
import into `pkg` tests. Add only the minimal local test support needed by each
module; shared utilities must justify their cost and avoid production refactors.

### First delivery batch: PRs 1–4

**PR 1 — Review question: does validator search remain a bound value?**

- [ ] Add the smallest SQL/arguments recorder or mock using the actual PostgreSQL
      GORM dialect, together with validator search tests as its first consumer.
- [ ] Exercise the real repository method and both result/count paths, including
      the count transaction and timeout statements where applicable.
- [ ] Compare normal and SQL-like search values in the same query branch;
      assert invariant SQL structure and intact search arguments.
- [ ] Keep the harness limited to this use case; no unrelated query rewrites.

**PR 2 — Review question: can sort input introduce a SQL expression?**

- [ ] Cover every supported validator sort and both boolean directions.
- [ ] Assert constant identifiers, uptime NULL ordering, and secondary sorts.
- [ ] Verify empty/unknown/SQL-like sort values keep the existing voting-power
      fallback rather than introducing a new HTTP error response.

**PR 3 — Review question: can account filters introduce an identifier?**

- [ ] Exercise single and combined filters in result/count queries; only the
      seven existing literal columns may appear and selected values remain bound.
- [ ] Assert account IDs remain arguments and invalid hash search keeps its
      existing empty-result behavior without SQL execution.
- [ ] Compare allowed condition sets and parameter association, not exact OR
      ordering, because Go map iteration is nondeterministic.

**PR 4 — Review question: is the constructed module ID data in all three slots?**

- [ ] Exercise the actual raw stats method with SQL-like address/name values.
- [ ] Assert the query template is unchanged and the exact constructed module ID
      occupies all three bind arguments.

### Subsequent API coverage

**PR 5 — Review question: do proposal/NFT search and lists remain parameters?**

- [ ] Cover search, LIKE/regex patterns, and IN-list filters in result/count paths.
- [ ] Compare SQL structure for fixed branches and list sizes; preserve existing
      wildcard semantics, filtering, and empty-list behavior.

**PR 6 — Review question: is malformed pagination rejected before data access?**

- [ ] Send malformed limit, offset, reverse, and base64 cursor values through
      the HTTP boundary and assert no downstream repository invocation.
- [ ] Keep valid min/max limits, numeric cursors, and boolean behavior unchanged.
- [ ] Keep large-valid-offset performance work outside this SQL injection plan.

### Blockchain and shared-helper coverage

**PR 7 — Review question: does parsed memo remain data after normalization?**

- [ ] Add minimal DB recorder fixtures following PR 1's pattern.
- [ ] Exercise transaction parser/model mapping and the actual insert helper.
- [ ] Preserve quotes, backslashes, Unicode, and comment markers in bound values;
      separately assert NUL becomes U+FFFD. Normalization is not SQL escaping.

**PR 8 — Review question: do metadata and content JSON remain bound data?**

- [ ] Exercise proposal RPC mapping and actual proposal insertion, checking both
      the metadata field and its JSON representation as bound values.

**PR 9 — Review question: do event/NFT fields stay data through persistence?**

- [ ] Cover event attribute mapping and actual insert helpers, plus NFT/collection
      metadata updates. Values must never select SQL columns or expressions.

**PR 10 — Review question: does batching interpolate only placeholder numbers?**

- [ ] Cover validator address/image values with zero, one, two, and `BatchSize+1`
      records. Verify parameter association and numbering restart in each batch.

**PR 11 — Review question: are pruning identifiers limited to known tables?**

- [ ] Cover valid table names and reject unknown identifiers before SQL executes.
- [ ] Verify thresholds remain bound. Do not add a caller for `TruncateTable` or
      expand production allowlists as part of this coverage PR.

**PR 12 — Review question: does PostgreSQL store hostile-looking data literally?**

- [ ] Add a reproducible disposable PostgreSQL fixture and explicit setup,
      integration-test, and cleanup commands with build prerequisites.
- [ ] Round-trip memo and proposal metadata using production helpers, verifying
      expected exact values and unchanged unrelated control rows.
- [ ] Use harmless local payloads; no destructive/time-delay statements or
      production credentials. Report skipped integration tests separately.
- [ ] Keep this PR focused on two representative round trips rather than
      duplicating every recorder test as an integration test.

### Independent analysis setup

**PR 13 — Review question: is CodeQL coverage reproducible and triage grounded?**

- [ ] Inspect hosted/default setup and existing alerts before adding a workflow;
      avoid duplicate analysis and record permission limitations.
- [ ] Enumerate Go modules and include API, shared `pkg`, and relevant indexers
      with compatible build prerequisites and minimal workflow permissions.
- [ ] Record query suite/version, analyzed SHA, extraction/build outcome, and
      SARIF or hosted result links.
- [ ] Triage relevant alerts using source, transformation, sink, guards, attacker
      control, and reachability. Record blockchain source-modeling gaps; a clean
      standard scan alone does not prove blockchain input coverage.
- [ ] Compare equivalent base/head configurations before calling an alert a
      regression. Newly visible alerts may reflect older code or model changes.
- [ ] Keep any required custom source-model implementation or confirmed defect
      remediation in a separate follow-up PR per root cause. Do not enlarge this
      baseline PR to absorb those changes or suppress alerts just to pass CI.

Inspect hosted setup early even though delivery starts with PRs 1–4. PR 13's
estimate is less certain: existing setup may reduce work, while build failures
or a large alert backlog may require additional separately scoped work.

## Shared acceptance and validation

- Exercise existing repository/helper methods, not copied test-only queries.
- Inspect executable SQL templates and bound arguments separately; interpolated
  logs or `ToSQL` output do not establish parameter binding.
- For fixed branches/list sizes, changing a data value must not change SQL
  structure. Include both data and count queries where the method supports them.
- Preserve API responses, status codes, sort fallback, normalization, and existing
  wildcard behavior. No production/schema changes are required by this plan.
- Every PR must name the focused test command and demonstrate the relevant
  assertion catches an unsafe local mutation; do not commit that mutation.
- API PRs run affected packages in `api`; the relevant full gate is
  `go test ./dto ./handlers ./repositories ./services/...`. DB/chain PRs run
  affected packages in `pkg` and the owning indexer modules. PR 12 supplies its
  explicit PostgreSQL integration command; PR 13 supplies analysis evidence.
- If a reachable vulnerability is discovered, scope a separate fix and failing
  regression test. Bind values and use explicit allowlist mappings for SQL
  identifiers/directions; do not use manual string escaping as the primary fix.

## Safe local test corpus

Use only disposable fixtures or a SQL recorder:

```text
probe' OR '1'='1
probe'); SELECT 1; --
moniker DESC,(SELECT 1)
```

Also test quotes, comment markers, backslashes, Unicode, valid wildcard values,
and base64-encoded `0 OR 1=1` as a malformed cursor. A sort payload must trigger
the existing fallback, while data payloads must remain bind values. For exact
matches, an inserted literal payload should match only its own fixture row.

## Validation already performed and limitations

- `go test ./dto ./handlers ./repositories` in `api`: DTO and handler tests passed;
  repository package reported `[no test files]`.
- Existing service tests primarily mock repositories and do not prove SQL binding.
- No CodeQL executable was found on PATH; no local CodeQL/SARIF configuration or
  result artifacts were found in the repository. CodeQL was not run.
- Hosted CodeQL alerts/default setup were not inspected during the source review.
- No live PostgreSQL payload/round-trip tests or performance tests were run.
- No production code was changed during the review or by this planning document.

Review lenses: coordinator correctness review plus parallel security, test
coverage, and simplification passes. No confirmed exploitable finding was found;
SQL-boundary coverage gaps motivate this plan. General comment, error-handling,
and type-design audits were outside this focused review.
