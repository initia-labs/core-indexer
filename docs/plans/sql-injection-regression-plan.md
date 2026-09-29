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

## PR 1: API SQL-boundary regression tests

Suggested title: `test: verify API SQL parameter binding`

- [ ] Add a reusable SQL/arguments recorder or SQL mock using the actual PostgreSQL
      GORM dialect. Exercise existing repository methods, not copied queries.
- [ ] Cover search, list filters, module raw stats, account boolean filters,
      validator sorting, pagination, and both result and count execution paths.
- [ ] For each fixed query branch and list size, compare a normal value with a
      SQL-like value. Assert invariant SQL structure and intact bound arguments.
- [ ] Assert module IDs occupy all three arguments in the raw stats query.
- [ ] Test each supported sort and both directions, including uptime NULL ordering
      and secondary sorts. Unknown sort must retain the voting-power fallback.
- [ ] Assert only the seven account-filter identifiers are selectable. Account
      for nondeterministic Go map iteration instead of snapshotting OR order.
- [ ] Test malformed limit, offset, reverse, and base64 cursor values at the HTTP
      boundary; prove rejection occurs before repository/database invocation.
- [ ] Preserve invalid hash-search empty results and existing wildcard semantics.

Acceptance: SQL-like data cannot become SQL syntax; valid API response behavior,
status codes, sort fallback, and pagination semantics remain unchanged. Cover
count queries as well as result queries. Inspect driver SQL and args separately;
interpolated logs or `ToSQL` output do not establish binding safety.

Validation: run `go test ./dto ./handlers ./repositories ./services/...` in `api`.
A deliberately temporary local mutation replacing binding with interpolation
should make the relevant test fail; do not commit that mutation.

## PR 2: Blockchain and shared-helper regression tests

Suggested title: `test: verify blockchain data remains SQL parameters`

- [ ] Reuse the recorder pattern to exercise parser/model mapping and actual DB
      helpers for memo, proposal metadata/content JSON, event attributes, and
      NFT metadata. Preserve quote, backslash, and Unicode data.
- [ ] Test memo NUL normalization independently; expected stored data replaces
      NUL with U+FFFD while retaining SQL metacharacters as literal data.
- [ ] Cover validator image updates with zero, one, two, and `BatchSize+1` rows;
      verify parameter association and placeholder numbering for every batch.
- [ ] Verify pruning accepts its known tables and rejects unknown identifiers
      before any SQL executes.
- [ ] Add isolated local PostgreSQL round-trip fixtures with SQL-like strings
      and unrelated control rows; verify exact stored values and unchanged
      control rows. Avoid destructive or time-delay payloads.
- [ ] Document a reproducible disposable database setup and cleanup. Never use
      a production database or derive credentials from deployment configuration.

Acceptance: the production helpers keep supplied values out of SQL structure;
round-trip storage preserves expected data. No schema or API changes required.

Validation: focused tests in `pkg/db`, `pkg/txparser`, and relevant indexer
mapping packages, followed by the explicit PostgreSQL integration test command
introduced by this PR. Document build prerequisites and separate skipped
integration tests from passing tests.

PRs 1 and 2 may proceed independently. Share test utilities only when that makes
both test suites smaller; do not require a production query-builder rewrite.

## PR 3: CodeQL baseline and alert triage

Suggested title: `ci: establish Go CodeQL baseline`

- [ ] Inspect hosted/default CodeQL setup and existing alerts before adding a
      workflow, avoiding duplicate analysis. Record any permission limitations.
- [ ] Enumerate the repository Go modules and ensure analysis includes API,
      shared `pkg`, and relevant indexers with compatible build prerequisites.
- [ ] Capture query suite/version, analyzed SHA, extraction/build outcome, and
      SARIF or hosted result links. Use minimal required workflow permissions.
- [ ] Triage each relevant alert with source, transformation, sink, validation,
      attacker control, and reachability evidence. Blockchain inputs may require
      extra source modeling; a clean standard scan alone does not prove coverage.
- [ ] Establish a baseline before making regression claims. Compare equivalent
      analysis configurations on base/head; a newly visible alert may be old code
      or a query/model/configuration change.
- [ ] If an exploitable path is confirmed, create a narrowly scoped follow-up
      remediation PR with a failing regression test. Bind data values; map
      identifiers and sort directions through explicit allowlists. Preserve API
      behavior and do not make manual escaping the primary fix.

Acceptance: reproducible analysis and evidence-backed dispositions. Do not
suppress alerts simply because a helper uses GORM or because tests pass.
This PR can run independently of PRs 1 and 2.

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
