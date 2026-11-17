"""Independent contract tests; execute source bytes, never a local import/cache."""
from dataclasses import asdict, replace
from functools import lru_cache
import hashlib
import pathlib
import subprocess
import tempfile
import sys
import types
import unittest
from unittest import mock

SOURCE = pathlib.Path(__file__).with_name("check.py")


def load_source(path=SOURCE):
    module = types.ModuleType("outing_model_test_subject")
    module.__file__ = str(path)
    sys.modules[module.__name__] = module
    exec(compile(path.read_bytes(), str(path), "exec"), module.__dict__)
    return module


M = load_source()


class InitialControls(unittest.TestCase):
    def test_late_public_authorization_is_denied(self):
        self.assertEqual(M.authorization_disposition(True, "", "invocation:weather:assess"), "deny")

    def test_terminal_answer_with_forecast_debt_is_not_completion(self):
        self.assertFalse(M.completion_allowed(("recommended", 0), ("forecast:0:selection",), (True, True)))



@lru_cache(maxsize=None)
def graph(variant="delegated", outcomes=(), allowance=1, domain="broad", no_result=()):
    return M.explore(M.Config(variant, outcomes, allowance, domain, no_result))


def take(config, state, stage, operation=None):
    options = [(ev, nxt) for ev, nxt in M.successors(config, state)
               if ev.stage == stage and (operation is None or ev.operation == operation)]
    if len(options) != 1:
        raise AssertionError((stage, operation, options))
    ev, nxt = options[0]
    M.validate_transition(config, state, ev, nxt)
    return ev, nxt


class ProtocolTests(unittest.TestCase):
    def test_broad_initial_completion_but_nonblocking_counterexample(self):
        for variant in ("direct", "delegated"):
            g = graph(variant)
            self.assertIn(0, g.completing)
            self.assertLess(len(g.completing), len(g.states))
            summary = M.graph_summary(g)
            trace = summary["counterexamples"]["source_refusal"]["trace"]
            self.assertEqual(summary["counterexamples"]["source_refusal_acknowledged"]["trace"][-1]["stage"], "message.acknowledge")
            self.assertTrue(summary["counterexamples"]["allow_before_end_admit_after_end"]["has_successful_suffix"])
            self.assertEqual(trace[-1]["disposition"], "deny")
            self.assertEqual(trace[-2]["stage"], "root.end_freeze")
            self.assertEqual(trace[-3]["operation"], "message:outing:decision")
            blocked = summary["counterexamples"]["nonblocking"]["state"]
            self.assertTrue(all(i in g.completing for i in range(blocked)))

    def test_preclose_domain_nonblocking_known_two_candidates(self):
        for variant in ("direct", "delegated"):
            g = graph(variant, ("unsuitable", "suitable"), domain="preclose-authorization")
            self.assertEqual(len(g.completing), len(g.states))
            self.assertEqual({g.states[i].decision for i in g.marks}, {("recommended", 1)})
            self.assertEqual({g.states[i].charges for i in g.marks}, {(int(variant == "delegated"), int(variant == "delegated"), 2)})

    def test_known_unavailable_is_diagnostic_not_success(self):
        g = graph(outcomes=("unavailable",), domain="preclose-authorization")
        self.assertFalse(g.marks)
        self.assertTrue(g.diagnostics)
        for i in g.diagnostics:
            self.assertEqual(g.states[i].helper, "succeeded")
            self.assertEqual(g.states[i].weather, "failed")
        for classes, answer in ((("unavailable", "suitable"), ("recommended", 1)),
                                (("suitable", "unavailable"), ("recommended", 0))):
            self.assertEqual(M.policy(classes), answer)

    def test_refusal_keeps_caller_debt_but_no_descendant_or_charge(self):
        g = graph(allowance=0)
        self.assertFalse(g.marks)
        for state in g.states:
            if state.invocation.admission == "refused":
                self.assertEqual(state.charges, (0, 0, 0))
                self.assertEqual(state.helper, "absent")
                self.assertEqual(M.root_debts(state, 1), ())
                self.assertIn("outing:assessment_receive", M.debts(g.config, state))
                self.assertFalse(state.invocation.result_copy)
        published = next(s for s in g.states if s.invocation.published and not s.invocation.selected)
        self.assertTrue(published.invocation.outcome_delivered)
        self.assertIn(M.CALL + ":selection", M.debts(g.config, published))
        self.assertTrue(any(s.weather == "failed" and s.root_publications[1] for s in g.states))

    def test_no_result_blocks_original_without_retry_or_later_work(self):
        g = graph(outcomes=("suitable", "unsuitable"), no_result=(0,))
        self.assertFalse(g.marks)
        unknown = next(s for s in g.states if s.forecasts[0].stage == 4)
        self.assertEqual(unknown.forecasts[1].stage, 0)
        self.assertEqual(unknown.contacts, 1)
        self.assertFalse(any(ev.operation == unknown.forecasts[0].operation for ev, _ in M.successors(g.config, unknown)))
        self.assertTrue(all(s.forecasts[1].stage == 0 for s in g.states))

    def test_missing_result_cut_also_has_known_result_continuation(self):
        pending = graph(outcomes=("suitable",), domain="preclose-authorization")
        interrupted = graph(outcomes=("suitable",), domain="preclose-authorization", no_result=(0,))
        cut = next(s for s in interrupted.states if s.forecasts[0].stage == 4)
        index = pending.states.index(cut)
        self.assertIn(index, pending.completing)
        self.assertFalse(interrupted.completing)
        self.assertTrue(any(ev.stage == "forecast.result" for ev, _ in M.successors(pending.config, cut)))
        self.assertFalse(any(ev.stage == "forecast.result" for ev, _ in M.successors(interrupted.config, cut)))

    def test_confirm_first_fixed_two_candidate_cycle_has_zero_contact(self):
        g = graph("confirm-first", ("suitable", "unsuitable"))
        self.assertFalse(g.marks)
        self.assertFalse(g.diagnostics)
        self.assertTrue(all(s.contacts == 0 and s.charges == (0, 0, 0) for s in g.states))
        self.assertIn("confirm_first_cycle", M.graph_summary(g)["counterexamples"])

    def test_allow_before_end_admit_after_end_and_idempotent_retry(self):
        g = graph("direct")
        before = next(s for s in g.states if s.outing == "ready_end" and not s.messages[2].authorization)
        authorization, state = take(g.config, before, "message.authorize", "message:outing:decision")
        _, state = take(g.config, state, "root.end_freeze", "root:outing")
        self.assertEqual(M.repeat_service(g.config, state, authorization), state)
        _, state = take(g.config, state, "message.admit", "message:outing:decision")
        self.assertEqual(state.messages[2].admission, "accepted")
        self.assertIn(state, [g.states[i] for i in g.completing])

    def test_deny_after_end_is_immutable_and_ack_is_not_consumption(self):
        g = graph("direct")
        state = next(s for s in g.states if M.terminal(s.outing) and not s.messages[2].authorization)
        denied, state = take(g.config, state, "message.authorize", "message:outing:decision")
        self.assertEqual(denied.disposition, "deny")
        admission, state = take(g.config, state, "message.admit", "message:outing:decision")
        ack, state = take(g.config, state, "message.acknowledge", "message:outing:decision")
        for ev in (denied, admission, ack):
            self.assertEqual(M.repeat_service(g.config, state, ev), state)
        self.assertFalse(state.messages[2].selected)
        self.assertEqual(state.weather, "await_decision")
        with self.assertRaises(ValueError):
            M.repeat_service(g.config, state, replace(denied, disposition="allow"))

    def test_bound_reply_after_end_freeze_waits_for_admission(self):
        g = graph()
        state = next(s for s in g.states if s.helper == "succeeded" and not s.messages[3].authorization)
        self.assertIn(state.messages[3].operation, state.invocation.frozen)
        self.assertFalse(any(ev.stage == "invocation.publish_deliver" for ev, _ in M.successors(g.config, state)))
        authorization, state = take(g.config, state, "message.authorize", state.messages[3].operation)
        self.assertEqual(authorization.disposition, "allow")
        self.assertTrue(state.messages[3].source_ended_at_authorization)
        with self.assertRaises(ValueError):
            M.advance(g.config, state, replace(authorization, binding="another-invocation"))
        _, state = take(g.config, state, "message.admit", state.messages[3].operation)
        self.assertFalse(state.messages[3].selected)
        self.assertFalse(state.messages[3].acknowledged)
        _, state = take(g.config, state, "invocation.publish_deliver")
        self.assertTrue(state.invocation.result_copy and state.invocation.outcome_delivered)
        self.assertFalse(state.invocation.selected)
        self.assertIn(state.messages[3].operation + ":acknowledgment", M.debts(g.config, state))

    def test_atomic_publication_has_no_separate_delivery_cut(self):
        g = graph()
        self.assertTrue(all(s.invocation.published == s.invocation.outcome_delivered for s in g.states))
        published = next(s for s in g.states if s.invocation.published)
        with self.assertRaisesRegex(M.InvariantError, "atomic"):
            M.validate_state(g.config, replace(published, invocation=replace(published.invocation, outcome_delivered=False)))

    def test_mutated_transition_cannot_hide_debit(self):
        g = graph(outcomes=("suitable",))
        state = next(s for s in g.states if s.forecasts[0].stage == 2)
        ev, nxt = take(g.config, state, "forecast.reserve")
        bad = replace(nxt, charges=state.charges)
        with self.assertRaisesRegex(M.InvariantError, "ledger"):
            M.validate_state(g.config, bad)
        with self.assertRaisesRegex(M.InvariantError, "transition"):
            M.validate_transition(g.config, state, ev, bad)

    def test_same_answer_mutations_reject(self):
        g = graph(outcomes=("suitable", "unsuitable"), domain="preclose-authorization")
        good = g.states[min(g.marks)]
        bad_forecasts = (good.forecasts[0], replace(good.forecasts[1], stage=0, result=""))
        mutations = [replace(good, charges=(0, 0, 2)), replace(good, forecasts=bad_forecasts),
                     replace(good, invocation=replace(good.invocation, selected=False)),
                     replace(good, invocation=replace(good.invocation, binding="other")),
                     replace(good, forecasts=(replace(good.forecasts[0], operation="replacement"), good.forecasts[1])),
                     replace(good, invocation=replace(good.invocation, frozen=())),
                     replace(good, contacts=3)]
        for bad in mutations:
            self.assertEqual(bad.decision, good.decision)
            with self.subTest(bad=bad), self.assertRaises(M.InvariantError):
                M.validate_state(g.config, bad)
        # A valid state with the same answer still owes transport acknowledgment.
        unacked = M.message_update(good, 3, acknowledged=False)
        M.validate_state(g.config, unacked)
        self.assertFalse(M.successful_mark(g.config, unacked))
        with self.assertRaisesRegex(M.InvariantError, "hides obligations"):
            M.validate_mark(g.config, unacked, True)

    def test_bad_authorization_transition_is_detected(self):
        g = graph("direct")
        before = next(s for s in g.states if M.terminal(s.outing) and not s.messages[2].authorization)
        ev, after = take(g.config, before, "message.authorize", "message:outing:decision")
        bad = M.message_update(after, 2, authorization="allow")
        with self.assertRaisesRegex(M.InvariantError, "disposition"):
            M.validate_state(g.config, bad)
        with self.assertRaises(M.InvariantError):
            M.validate_transition(g.config, before, ev, bad)

    def test_initial_state_cannot_be_marked_successful(self):
        cfg = M.Config("direct", ())
        with self.assertRaises(M.InvariantError):
            M.validate_mark(cfg, M.initial(cfg), True)

    def test_direct_remaining_budgets_do_not_invent_helper_capacity(self):
        cfg = M.Config("direct", (), 1)
        view = M.state_view(cfg, M.initial(cfg))
        self.assertEqual(view["remaining"]["helper_invocations"], 0)
        self.assertEqual(view["remaining"]["created_runs"], 0)

    def test_nonempty_different_answers_are_not_successful_replacement(self):
        def divergent_summary(config):
            return {"config": asdict(config), "semantic_answers": [("recommended", int(config.variant == "delegated"))],
                    "successful_costs": [], "states": 1, "edges": 0}
        with mock.patch.object(M, "explore", side_effect=lambda config: config), mock.patch.object(M, "graph_summary", side_effect=divergent_summary):
            result = M.enumerate_models()
        self.assertTrue(all(not pair["semantic_answer_sets_equal"] for pair in result["comparisons"]))
        self.assertTrue(all(not pair["successful_replacement"] for pair in result["comparisons"]))

    def test_inputs_and_unknown_events_are_closed(self):
        for arguments in (("other", ()), ("direct", ("suitable",) * 3), ("direct", ("unknown",)),
                          ("delegated", (), 2), ("direct", (), 1, "fair"), ("direct", (), 1, "broad", (0,))):
            with self.subTest(arguments=arguments), self.assertRaises(ValueError):
                M.Config(*arguments)
        cfg = M.Config("direct", ())
        with self.assertRaises(ValueError):
            M.advance(cfg, M.initial(cfg), M.Event("invented", "x", "outing"))


class LauncherTests(unittest.TestCase):
    def test_exact_captured_bytes_execute_after_both_paths_change(self):
        launcher = load_source(SOURCE.with_name("run.py"))
        with tempfile.TemporaryDirectory() as directory:
            output = pathlib.Path(directory) / "result.json"
            paths = {name: pathlib.Path(directory) / name for name in launcher.SOURCES}
            captured = {
                "check.py": ("import json\nfrom pathlib import Path\ndef main():\n    Path(" + repr(str(output)) + ").write_text(json.dumps(dict(EXECUTION_PROVENANCE, marker=EVIDENCE_API['marker'])))\n").encode(),
                "evidence.py": b"marker = 'captured evidence'\n",
            }
            for path in paths.values():
                path.write_text("raise RuntimeError('path was reread')\n")
            launcher.execute_sources(captured, paths, b"launcher bytes", ["test-command"])
            import json
            result = json.loads(output.read_text())
            self.assertEqual(result["marker"], "captured evidence")
            hashes = {name: hashlib.sha256(raw).hexdigest() for name, raw in captured.items()}
            self.assertEqual(result["source_files"], hashes)
            self.assertEqual(result["source_sha256"], hashes["check.py"])
            identity = b"outing-source-set/v1\x00" + json.dumps(hashes, sort_keys=True, separators=(",", ":")).encode()
            self.assertEqual(result["source_set_sha256"], hashlib.sha256(identity).hexdigest())
            self.assertEqual(result["launcher_sha256"], hashlib.sha256(b"launcher bytes").hexdigest())

    def test_missing_or_unexpected_executable_source_rejects(self):
        launcher = load_source(SOURCE.with_name("run.py"))
        for names in (("check.py",), ("check.py", "evidence.py", "extra.py")):
            with self.subTest(names=names), self.assertRaisesRegex(ValueError, "exact declared executable source set"):
                launcher.execute_sources({n: b"raise RuntimeError('must not execute')" for n in names}, {n: pathlib.Path(n) for n in names}, b"launcher", [])

    def test_isolated_launcher_main_uses_supported_interpreter_api(self):
        with tempfile.TemporaryDirectory() as directory:
            launcher = pathlib.Path(directory) / "run.py"
            launcher.write_bytes(SOURCE.with_name("run.py").read_bytes())
            source = b"import json\ndef main():\n    print(json.dumps(EXECUTION_PROVENANCE))\n"
            launcher.with_name("check.py").write_bytes(source)
            launcher.with_name("evidence.py").write_bytes(b"pass\n")
            result = subprocess.run([sys.executable, "-I", "-B", str(launcher)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            import json
            provenance = json.loads(result.stdout)
            self.assertEqual(provenance["source_sha256"], hashlib.sha256(source).hexdigest())
            self.assertEqual(set(provenance["source_files"]), {"check.py", "evidence.py"})
            self.assertTrue(provenance["isolated"] and provenance["bytecode_disabled"])
            launcher.with_name("evidence.py").unlink()
            missing = subprocess.run([sys.executable, "-I", "-B", str(launcher)], capture_output=True, text=True)
            self.assertNotEqual(missing.returncode, 0)

    def test_unsupported_cli_and_nonisolated_execution_fail(self):
        launcher = str(SOURCE.with_name("run.py"))
        for command in ([sys.executable, "-I", "-B", launcher, "unknown"], [sys.executable, launcher, "enumerate", "--output", "unused"]):
            result = subprocess.run(command, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
        result = subprocess.run([sys.executable, "-I", "-B", str(SOURCE), "enumerate", "--output", "unused"], capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unbound source execution", result.stderr)


if __name__ == "__main__":
    unittest.main()
