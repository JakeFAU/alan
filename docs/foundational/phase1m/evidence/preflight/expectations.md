# C_se preflight expectations — recorded before execution

Question: does an ordinary single owning parent consume original S before the
candidate while preserving the strict guarded original-outcome join?

Specification-derived preflight: id1/value0/mode answer for original A and B.
Candidate response must be present, succeeded, id1/value1. Original S must receive
id1/value0 (not the complement), and return present succeeded id1/value0. Both
original invocations commit e then S before the first await, and their original
outcomes must each be selected and consumed exactly once, S then e. Parent and
root/aggregate publication must succeed. One fixed parent, I=R3, depth2, E0,
physical capacity1. Expected dynamic I/R spending: A+S2; B+helper+S3. Different
actual debits are permitted under the common worst-case entitlement.

Controls: candidate failure/absence and S failure/absence must consume both
original outcomes before Failure without reading absent response fields.
Candidate pending permits S consumption then leaves parent open at candidate
await; S pending leaves parent open at first await even if candidate completed.
No second request is supplied. Wrong-id is an honest invalid-component control.

These expectations are from assignment sections2–3 and K1's finite request
contract. Automatic execution observations are construction feasibility evidence,
not an independent model/oracle definition or exhaustive substitution result.
Stable generations/revisions/grants, valid bindings/schemas, no cancellation or
retirement and eventual eligible service are assumptions. Operational 200-step/
200-dispatch bounds must be checked against compiled code and actual reports.
