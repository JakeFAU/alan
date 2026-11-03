# Inspect a retained execution

`inspect-execution STORE` reads a single coherent execution snapshot. Each run lists its own invocation and effect operations. A published outcome includes the original invocation ID and the parent reduction that selected it, when selection has occurred. The resource line separates invocation charges, created runs and effect reservations. Effects show their source authority, local execution bounds, original settlement and any recorded usage. Their consequence field describes the retained evidence of external application. A local failure can leave that consequence unresolved.

## Completed investigation

Start the [reading-room fixture](../investigator/README.md#run-the-offline-example) in one terminal. In another terminal, from the repository root, run the following commands. The inspector requires only the database. You can stop the fixture after the investigation completes.

```sh
work=$(mktemp -d)
go build -o "$work/investigator" ./examples/investigator
go build -o "$work/inspect-execution" ./examples/inspect-execution
db="$work/reading-room.sqlite"
"$work/investigator" init "$db" examples/investigator/testdata/case.json
"$work/investigator" run "$db"
"$work/inspect-execution" "$db"
```

The completed fixture includes these lines from different parts of the summary.

```text
root "investigator" generation=1 local=succeeded published_outcome=succeeded invocations=2/2 created_runs=2/2 effect_reservations=4/4 local_capacity=2
  authored_completion="disagreement"
    evidence document="inspection" sha256=771a445d5806b817da6208739aee8de44743fd48bd1c0f3c58934b038a29c27b bytes=[0,44) matches_source=true quote="The ventilation inspection passed on Monday."
```

The two invocation operations publish separate worker outcomes. Their lines report `parent_selected_sequence=5` and `parent_selected_sequence=6` in this example. The two model results each report 30 input tokens, 20 output tokens and 50 total tokens. These values come from the controlled HTTP fixture. Missing counters render as `?`, and a missing usage object renders as `reported_usage=unknown`.

## Failed and interrupted workers

The following qualification builds the same command in a separate Go module, runs controlled investigators, and checks the command’s output and absence of new provider contacts.

```sh
go test -count=1 -v -run '^TestExternalExecutionInspector$' ./examples/investigator
```

* A known provider refusal produces `retained_disposition="refusal"` on the original model effect. The worker publishes a failed outcome. Both worker outcomes are selected and the investigator retains `authored_completion="partial"`.
* In the interrupted case, the HTTP fixture acknowledges the counter worker’s request and holds its response. At the primary worker’s acknowledged publication boundary the parent has yet to select that outcome. The inspector reads the database while the owner is paused, after an actual SIGKILL, and after another process reopens the store. The pending model line includes the **same original operation ID**, `physical_executions=0..1`, `reported_usage=unknown` and `remote_outcome=unknown retry=not-automatically-authorized`. The independent server saw the POST. The retained reservation alone cannot establish whether it reached the provider. The original charge and pending obligation survive.
* Local run status, publication, and selection are independent fields. A published child can be unconsumed while its parent waits for another original outcome. Quiescence describes local eligibility. The provider may still be processing work from the previous owner.

The command registers no adapters and performs no replay. It reports recorded provider fields as evidence from the registered adapter. The investigator’s `answer` command additionally applies its deployment, proposal and HTTP-body consistency policies. The reader independently checks the example lookup snippet’s SHA-256 version and half-open UTF-8 byte range against the retained asset. A source match authenticates the quotation. Interpreting that quotation remains application work. No monetary cost is derived from token counts.

## What moved out of the application

At the merged Milestone 1 baseline, the investigator’s answer and deployment paths issued no SQL themselves, but directly named twelve retained tables at fifteen call sites. They used twelve row searches and twelve per-descriptor JSON decode call sites across those files. They manually followed admissions to children, outcomes to delivery observations, selections to exact reductions, and effect attempts to results.

Tracing the completed 76-step reading-room fixture measured 1,122 SQLite statements inside the original answer command and 55 in the migrated command. Both produced the same 5,571-byte report.

The migrated paths issue zero SQL, name zero tables and decode zero storage descriptors. They use one typed cut per decision and the indexed run, invocation, effect and asset accessors. A fresh owner still rechecks its spending approval under its owner lock. The example retains the application checks that make its answer trustworthy, including its expected roles and programs, provider configuration and response evidence, and exact document byte spans. [Investigator](../investigator/README.md)
