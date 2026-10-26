# Phase1F live controller independent review

Reviewed 2026-10-26. Scope: the two files in `live-control-review.patch`, their existing boundary/adapter/CLI dependencies, prior retained live evidence, and the supplied local pricing assumption. No live calls, socket connections, credentials, `.env` reads, subagents, product edits, or commits were used. Only this report was added. The original retained snapshot was read without modification. This review does not independently revalidate current provider prices or claim an invoice/hard billing guarantee.

**Specification verdict: changes required. Quality verdict: changes required.** One P2 correctness finding prevents approval for paid evaluation. Reservation durability, descriptor handoff, causal correlation, absent-result reservation retention, and replay setup otherwise passed source inspection and the stated offline checks.

## Finding

**P2 — Retain the reservation when usage totals are contradictory or invalid.** `docs/foundational/phase1f/tools/live_control.py:154–170` only checks that all three token counts are present; its nonnegative check excludes `TotalTokens`, and it never checks `TotalTokens == InputTokens + OutputTokens`. A returned usage object `{InputTokens: 0, OutputTokens: 0, TotalTokens: -1, Details: {}}` converts a $0.20 reservation into `status="usage-known", budget_charge_usd=0`. Counts `1/1/666` similarly reduce it to $0.00001875 despite contradictory usage. Those freed funds are available to later controlled runs through `charge()` and `add_operation()`. The existing adapter decodes these fields as integer pointers without validating the relationship (`adapters/openai/openai.go:201–216`), and retained result validation checks identity, digest, response bounds, and schema rather than usage totals (`internal/executor/adapters.go:413–442`), so this is reachable with malformed provider metadata rather than only an impossible synthetic inspection.

Require nonnegative integer counts and a consistent total before reducing a reservation; validate cache-detail counts as nonnegative integers as well. On malformed or contradictory usage retain the full reservation and identify the unresolved cost. Add negative-total and inconsistent-total regression cases. No application changes are needed. This finding concerns the controller's conservative accounting assumption; no local controller can verify whether a provider's internally consistent usage statement is truthful.

## Other reviewed behavior

- `write_json()` flushes/fsyncs the temporary file, atomically replaces the destination, then fsyncs its directory. Under the one-owner ledger lock, the operation is appended and persisted before boundaries evidence is written and before `os.write(rel_w, b"c")`. An exception before release cannot authorize transport; an exception after the durable reservation leaves the reservation in the ledger. Missing results retain the full reserve. A reconciliation/inspection exception leaves the prior durable reservations in place, although final run evidence may then remain incomplete.
- The barrier commits adapter authorization before emitting the pre-transport event, and `transportAdapter` calls `Execute` only after the barrier returns. A selected barrier treats EOF as error. The controller checks PID/store, retained intent and authorization IDs, run identity, epoch, sequence, current run epoch, and one-shot token. Duplicate operation IDs cannot be released twice. Seventeen accepted events and sixty-four one-shot filters cover the supplied bounded application; an eighteenth selected event is rejected before release. The controller supplies only the release byte, never provider content or ordinary program decisions.
- Existing OpenAI transport has one `client.Do`, no redirect following, a fresh non-reused transport, and a non-rewindable request body. Lookup is local retained-byte processing. Endpoint/model/output ceiling and default tier are checked before model release. The byte-size + 8192 overhead, maximum input/cache-write price, and 25% margin are explicitly local assumptions, not provider billing enforcement.
- With the held ledger lock occupying fd3, the actual descriptor allocation was ack read/write 4/5 and release read/write 6/7. The unchanged wrapper successfully duplicated fd5 to fd3 and fd6 to fd4, transmitted an acknowledgment, accepted one release byte, and exited zero. `pass_fds` carries only the child acknowledgment writer/release reader; it does not inherit the parent's release writer or ledger lock.
- Credential parsing uses `shlex.split` rather than shell evaluation. The value is passed through the child environment, not arguments or retained prepared requests. Inspection/replay remove `OPENAI_API_KEY` and `ALAN_BARRIER`; replay builds no adapter registry and is configured with network denial plus a positive control and inspection equality. The output scan checks literal, JSON-escaped, URL-encoded, and base64 key forms. No real key or credential source was read to assess this path. The replay and credential scan are after execution; their failure prevents the final successful summary, while already durable spend reservations remain.

## Verification and original probe inputs

The supplied suite passed all nine tests, using:

```sh
cd docs/foundational/phase1f/tools
env -u OPENAI_API_KEY PYTHONDONTWRITEBYTECODE=1 python3 -m unittest -v test_live_control
```

The following additional original probe ran from the repository root. It mutates only deep copies of the retained snapshot in memory; temporary descriptor artifacts are removed automatically. It makes no network calls and supplies no credential. Both malformed usage cases were accepted, demonstrating the finding. The fd probe returned acknowledgment `ack\n`, exit 0, stdout `fd-release-ok\n`, and empty stderr.

```python
import sys,json,copy,os,tempfile,subprocess
from pathlib import Path
sys.path.insert(0,'docs/foundational/phase1f/tools')
import live_control as c
v=json.loads(Path('docs/foundational/phase1f/evidence/live-phase1e/inspect.json').read_text())
a=json.loads(v['Snapshot']['Tables']['adapter_attempts'][0]['descriptor'])
for name,usage in [('negative-total',{'InputTokens':0,'OutputTokens':0,'TotalTokens':-1,'Details':{}}),('inconsistent-total',{'InputTokens':1,'OutputTokens':1,'TotalTokens':666,'Details':{}})]:
    view=copy.deepcopy(v)
    candidate={'ID':'probe','ReturnedModel':c.MODEL,'Response':'{"service_tier":"default"}','Usage':usage,'RequestID':'probe','ResponseID':'probe'}
    view['Snapshot']['Tables']['adapter_results']=[{'descriptor':json.dumps({'Candidate':candidate})}]
    run={'operations':[{'id':'probe','reserved_usd':.2}]}
    c.reconcile(run,view)
    print(name,json.dumps(run), 'charge=',c.charge({'controlled_runs':[run]}))
with tempfile.TemporaryDirectory(prefix='alan-live-review-fd-') as d:
    with (Path(d)/'ledger.lock').open('a') as lock:
        ar,aw=os.pipe();rr,rw=os.pipe()
        wrapper='import os,sys; os.dup2(int(sys.argv[1]),3); os.dup2(int(sys.argv[2]),4); os.execv(sys.argv[3],sys.argv[3:])'
        child='import os; os.write(3,b"ack\\n"); assert os.read(4,1)==b"c"; print("fd-release-ok")'
        proc=subprocess.Popen([sys.executable,'-c',wrapper,str(aw),str(rr),sys.executable,'-c',child],pass_fds=(aw,rr),stdout=subprocess.PIPE,stderr=subprocess.PIPE,env={'PATH':os.defpath})
        os.close(aw);os.close(rr)
        ack=os.read(ar,64);os.write(rw,b'c');out,err=proc.communicate(timeout=5)
        print('fd-probe',json.dumps({'fds':[ar,aw,rr,rw],'ack':ack.decode(),'exit':proc.returncode,'stdout':out.decode(),'stderr':err.decode()}))
        os.close(ar);os.close(rw)
```

Observed budget charges: `negative-total -> 0.0`; `inconsistent-total -> 0.00001875`. Each began with `reserved_usd=0.2` and ended `usage-known`.

No complete paid-controller run, live provider behavior, or actual replay sandbox invocation was attempted in this review. Failure ordering beyond the local probes was inspected in source; the nine tests are helper-level tests rather than full process/crash tests.

## Exact reviewed SHA-256 hashes

```text
1f7d8796fa9d37a015e78fa68310fc6fce1c51e94b3bf9332268efe42e87e682  docs/foundational/phase1f/tools/live_control.py
e230162d4cd44c10080567f019376be62212bdfafaa60e73f4ca00c962e15a5b  docs/foundational/phase1f/tools/test_live_control.py
3ced94eaa9c081a33b9a5ff3cbf01aee48e1eb49a6e3075d0d06336d23b0ae3d  internal/boundary/boundary.go
e0b618c59b939d1f89ac27f50e65d6de23ba10c641c9c687a3367991a237118f  internal/executor/adapters.go
3d27c76a1aeb8c7b8afe3d2ca4e79b973fabb687f39f479fab987b147dec329a  adapters/openai/openai.go
93cb8191306dd45b01f9bc3c9fea1822b06c9fd292133265a4695ebcbe80ee6f  adapters/lookup/lookup.go
1f8a8cbe17efdbbf662dcc7a5436878b2a699d61efecd4d4fe70623a07ae22c7  effects/effects.go
896ceed8d55dbb6c99525c70c63bea0eae3c1b98c85d49fa0be101d00079e290  examples/evidence/store.go
327b45abcfe76cd72cc15b5cf812bb9b5b5da995a3905960b2c040638eb42ec1  examples/evidence/evidence.go
491398f783968de61bc9d58c18b922bc723c9fb0e6c38dce2d78d8bb1e4f94f5  cmd/alan-evidence/main.go
eab6c7c7cd120157927b0e4496038ba42580357cf82ed7113c8ec21221c60a80  examples/investigation/store.go
2212ab5f78fa862244792048a3a9f5ffb8f1770427807c9d1cf0d22c419600ce  examples/investigation/investigation.go
c7024f0a5b96eb904262732b88feb43c788ce5da7d9d6a2b53d9522cc871ba08  cmd/alan-investigate/main.go
36c08abb4e6927aec3c56520ac40dbf926836a0f767a68cf35e2a4a5736e99b6  docs/foundational/phase1f/evidence/live-budget.json
0d0363aab04d5159eaafd61ad98738d75f779c6a420aa7c395009977b547f81b  docs/foundational/phase1f/evidence/live-phase1e/inspect.json
23dcde2331e8319ee3da523319254a8abf2618a4024fed98d8cdc46c0934b5b4  docs/foundational/phase1f/evidence/live-phase1e/result.json
769c364ff69621dc0d4c0fc7890b9f020a55fb8901c62b46286512bf3eb55cb2  .superpowers/sdd/00-plan/live-control-brief.md
14303b79fbfce8f585a3c3c0167c1b3d252b1462c585f92205298be9493244ad  .superpowers/sdd/00-plan/live-control-review.patch
```
