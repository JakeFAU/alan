# Independent reviews and delivery identity

These are separate agent reviews of actual source and retained evidence. They
provide scrutiny, not an unrestricted correctness proof or authorization to
merge. Each original review is copied byte for byte; `review-records.json`
binds its original name, byte count and SHA-256.

| Review | Reviewed head | Disposition |
|---|---|---|
| [Baseline and measurement](baseline-methodology.md) | `a8f32107aae32e5aa5a5785be1c6b91a139fbbeb` | Spec and quality APPROVE |
| [Report semantics and implementation](report-semantics.md) | `6fadae8575bb1567f4d11f69edbb8258d5f3dce6` | Spec and quality APPROVE |
| [Compact-result assessment](compact-assessment.md) | `63a85270eda04a5070e209715f65e77c3194c2c6` | Spec and quality APPROVE |
| [Qualification apparatus](qualification-apparatus.md) | `7679755c7298f229fa468b14da54bcfe1420c4f8` | Spec and quality APPROVE |
| [Frozen qualification and evidence](qualification-evidence.md) | `fd3a3fde1f1ddc4c397beaa84abedc63b03e1833` | Spec and quality APPROVE |
| [Whole branch](whole-branch.md) | `fd3a3fde1f1ddc4c397beaa84abedc63b03e1833` | Specification, quality and scoped branch APPROVE |

The earlier rejected SQL-presence designs, diagnostic-bypass witness, and
measurement-parser failures are preserved in [rejected variants](../rejected-variants.md)
with source labels and artifact hashes. Task approvals did not erase those
findings or substitute excluded runs for final measurements.

## Tested source and documentation-only delivery

The frozen tested source is `7679755c7298f229fa468b14da54bcfe1420c4f8`,
tree `7aeb84be8acf414800a649152746aacd3e03b5e6`. The evidence commit
`fd3a3fde1f1ddc4c397beaa84abedc63b03e1833` changes two README files and
adds results and evidence. It does not change executable source, tests,
fixtures, module files or consumed qualification tools. The independent
qualification review rehashed all 756 exported files and the 754 unchanged
live files, plus the selected tool, Go and SQLite inventories.

The final review-record commit adds only this review directory. The tests
retain their actual tested SHA; later documentation is linked by exact
file-content checks. [Delivery source bridge](source-bridge.json) records
the compared identities and permitted documentation paths. The PR records
the final delivery commit, which cannot be embedded in its own contents.

## Decisions and costs

- **Continue the authorized two-step slice without another permission round.**
  The user's “Make it so” authorized the proposed report batching and compact
  assessment. If that scope was misunderstood, the cost is a reviewable
  branch to revise; no merge or release was performed.
- **Retain this plan's workspace and earlier archives.** The user's evidence
  preservation requirement takes precedence over generic workspace cleanup.
  The cost is local disk use. Large raw artifacts remain explicitly local;
  the compact evidence and reproduction tools are committed.
- **Treat compact output as a conditional API decision.** Bypassing the full
  report loses an existing retained-history diagnostic. A compact API needs
  an explicit validation contract. Deferral leaves that developer/performance
  improvement open; the witness does not prove such an API impossible.
- **Stop the first implementation at root/run batching.** Defer additional
  acknowledgment and settlement batching to preserve a small review surface
  and an independently measured change. The cost is remaining per-intent
  queries and quadratic full-report work.
- **Review qualification tools before the confirmatory campaign.** Splitting
  apparatus review from execution added a gate rather than production scope.
  The cost was additional qualification time; it preserved the failed smoke
  attempts and kept final measurements on unchanged reviewed tools.

The native SDK and large-projection memory limitations remain as stated in
the [qualification](../qualification.md) and [results](../results.md).

## Final review boundaries

The controller accepts each of the final review's seven boundaries as a
limit of the delivered claim, not an exemption from a failed requirement:

| Boundary | Controller disposition |
|---|---|
| Arbitrary histories and large projections | Claim only the measured seven-bound series. Raw projection bytes add temporary memory; other workloads need their own measurement. |
| Wall-clock cancellation and private SQL callbacks | Preserve the supported cancellation causes and reached column-denial behavior. Identical query timing/counts are incompatible with this optimization and are not public Store guarantees. |
| External SQL writers bypassing ownership | Rely on existing owner exclusion and read transactions. No concurrent hostile-writer guarantee is added. |
| Stronger failure models | Qualify acknowledged local SIGKILL/reopen histories. Do not generalize them to power loss, device failure, distributed ownership or remote exactly-once effects. |
| Future compact APIs | Preserve the diagnostic counterexample and defer the API; feasibility and performance remain open. |
| Complete SDK and other hosts | Keep the stated selected-tool/SDK-settings binding and same-host assumption. No full SDK-byte or cross-host performance claim. |
| Bulky public artifacts | Keep the indexed raw archive locally and ship compact records and reproduction tools. No claim that local captures are publicly downloadable. |

If a consumer needs any stronger guarantee, this review does not supply it;
that requirement needs a separately specified and qualified change.
