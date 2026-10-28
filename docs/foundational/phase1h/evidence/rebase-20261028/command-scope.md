# New-main baseline command scope

Both commands ran against an isolated Git archive of56892cb20c4f4add4bcf786d146b0aed82fbc5d1. They are upstream baseline checks, not current Phase1H qualification.

The first default command passed author and internal/executor. Its runtime package reported `[no test files]` because runtime/runtime_test.go is integration-tagged; that command compiled runtime but did not execute its behavioral tests. The initial controller progress statement grouped all three as passing checks too broadly. The raw log and original command remain unchanged.

The follow-up command explicitly enabled `-tags integration` for runtime. It passed with53 passing test/subtest events, exit0, wall3.587s (package1.861s). Exact command, raw stdout/stderr hashes and package event are in baseline-integration-runtime-result.json. The earlier first-command exit0/7.918s remains separately attributed.
