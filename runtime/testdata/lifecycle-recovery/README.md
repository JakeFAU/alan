# Format 4 lifecycle recovery witnesses

The Milestone 2 writer at tree `9668a9f089b38bdb378e8b7eed69eea2e0f66295` produced both original schedules. Each started from a separate byte copy of the same closed, freshly provisioned database. The namespace, programs, target name, grants and initial authorities therefore agree. Both originals and both forgeries have the same final generation and epoch vector.

The SHA-256 digests below cover the uncompressed SQLite files. Each archive contains one 401,408-byte database. All committed frames are included in that main file. Tests unpack each case to a temporary directory before opening it.

| Archive | SHA-256 of database | Origin |
| --- | --- | --- |
| provisioned.sqlite.gz | `5f2f980e89660d323ea41257655f66b983a5b8c0e566242bed5a3033bfd4f1c9` | Pristine closed writer output, before either schedule |
| ready.sqlite.gz | `520c863359427cf1c71334308d070efe6d7d3edbdf7ab1fd7cd35efa8ec823a3` | Separate snapshot of the common setup on the missing branch, before the controller input. Captured by SQLite VACUUM INTO and returned to WAL mode for the process tests |
| missing.sqlite.gz | `82b6b15ce21d044e65d190283dc896c261487bbb196ed63b154172ec3d7d34b3` | Original missing lookup followed by creation |
| present.sqlite.gz | `bc3bdb93430c6fc64308ca5c5ec523816494df676d7029f32e262e120d0c6f7d` | Original witnessed lookup after creation |
| forged.sqlite.gz | `5d5880260b4d8e000e2bacf4a440d6b6479476b95c65b7cade280b71b751e23a` | Disposable copy of present.sqlite with the two changes below |
| forged-contiguous.sqlite.gz | `766d3a30bc730e08a00c69e43413f59df2c4fe02b2e8155f9ae4471fd188a0d1` | Disposable copy of forged.sqlite with the final execution fact renumbered from 8 to 7 |

## Executed schedules

1. On each branch, Open renewed the initial runs. The writer committed a non-idempotent external operation and retained an allowed reservation through the executor's real preparation transaction. It performed no HTTP exchange. The separate `finished` run committed its terminal instruction. The external operation remains unresolved in every later cut.
2. In **missing**, AdmitInput accepted `control-target`. The controller selected its observation in reduction 1, emitted CancelRun in reduction 2, obtained source authorization and received a durable `target_missing` decision with no before or after witness. The root then emitted and admitted the invocation. A separate input named `presence` to its entry was refused with a non-null binding to generation 1. The entry selected its original invocation request in reduction 1.
3. In **present**, the root emitted and admitted the same invocation and `presence` obtained its non-null binding **before** `control-target` entered the admissions journal. The controller selected its input, emitted the same CancelRun, obtained authorization and received `grant_denied` with matching before and after witnesses of the open entry at sequence 0. The entry then selected the original invocation request in reduction 1.
4. **Forged** changes the canonical lifecycle result for the original operation to `Reason="target_missing"` with null `Before` and `After`. It also removes the target-only lifecycle fact `666902595bf75737b15dbfa5a81c7841d1b01e6ff31e718cd68c372953d772df` at order 7. The later execution fact still has order 8. All three APIs can reject this copy on the gap alone.
5. **Forged-contiguous** moves that final execution fact from 8 to 7 in both the order column and the canonical descriptor. Its identity and record digest stay the same. The resulting seven facts are contiguous. Two journals still contradict the missing lookup. In admissions, the refused `presence` input already bound to the entry precedes `control-target`. In observations, the entry's invocation request, written at creation, precedes the controller input. The controller consumed that input before emitting the lifecycle request.

`TestCausalRecoveryMissingWithContiguousFacts` checks copies of this last archive. InspectExecution, Replay and Open reject with either journal still proving presence. Exchanging only the two append positions in **both** journals removes both ordering witnesses. All three APIs accept that compatible copy with the original pending obligations, decisions, and final epochs before renewal. This last copy is a test mutation. The writer-captured genuine absence is `missing.sqlite.gz`.

## Retained identities

| Item | ID or state |
| --- | --- |
| Namespace | `c35964b3158d492fa0d69dff69892f31b21cee6696720c639b0f2b2adfca3859` |
| Resource root | `4e0bb65f09f4afee0eaefb056a1d89f2aee1d1ada5cbfecf98f3c0a494dd8a57` |
| Original lifecycle operation | `["retirement-controller",1,2,0]` from generation 1, revision 1. Source authorized at epoch 2, sequence 2. Target decision sequence 1. Settlement remains pending |
| Controller selection | `["observation","input","control-target"]` |
| Original caller operation | `["root",1,1,0]` |
| Invocation | `79a7f81b35486203d6086198432fc3072fda6e9f41782b769cd96500f2092d2e` |
| Entry and helper | `137e1522572b02966c069ece4545dbf67fb289781c9be8c144b0015c749ddecc` and `3a5f694a5e35b05738e808ffe3fb0d7a34d73c2122159c575410ae116a946456`, each open at generation 1 and epoch 1 |
| Original invocation request selection | `2d8761d3f1c96eb71b7754dd0152b8d687b19ec491791e4444c255a4ee31249c` |
| Outstanding external operation | `["writer",1,1,0]`, scope-bound charge of one effect. Key version 2, original namespace and operation ID |
| Remaining final epochs | `finished`, `retirement-controller`, `root` and `writer` each at generation 1 and epoch 2. Only `finished` has terminated |
| Resource account | One invocation, two created runs, one external effect. The root is open with a pending invocation and no published outcome |
