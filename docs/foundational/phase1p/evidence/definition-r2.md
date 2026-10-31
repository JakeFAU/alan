# Independent Task 1 definition rereview

**R1-1 ADDRESSED. Spec APPROVE. Semantic APPROVE.** This verdict covers the corrected definition freeze on base `d0def139632c73936a3e821d93271a62267fa46d`. Proposition P remains a proof target.

I inspected the line 52 correction and the original decision and admission rules. A committed, admission-undecided invocation with no first decision enables authorization even after source closure (`research/substitution/invocation_join.go:124–137`). The initial decision retains its ordered refusal or allowance. Matching descriptor lookup reuses the disposition (`internal/executor/messages.go:103–171`). Admission separately inspects the retained result and current source (`internal/executor/invocation.go:296–312`). These distinctions resolve the sole R1 finding. R1's scope and other conclusions carry forward.

SHA-256 of corrected definition `2f5b74ea71aea62faa157364e3c37b02b063d3ebb4115e77b6731838663e4d68`, one-row correction diff `41233479e5db2bce2fbb077c7f729a200d9e01306510520abc1883cd29ce317a`, appended author report `9d6baf0ddde1b4c22b9bfd08442832bbc2b2c65e32542ea13fc4d9255e53e39d`, preserved R1 review `24049142020983b8d4ff46965e797a420e4dabb7a788dd220dcb14211430d2f0`.
