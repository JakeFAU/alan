"""Independent finite Outing protocol, authored from the declared contract.

No application imports, compiler, reducer, generated graph, or Store verifier.
Task 4 may add raw-record readers below the protocol API; this module is source-only.
"""
import argparse
from collections import deque
from dataclasses import asdict, dataclass, replace
import itertools
import json
import platform
import sqlite3

VERSION = "outing-protocol-model/v1"
DOMAINS = ("broad", "preclose-authorization")
CLASSES = ("suitable", "unsuitable", "unavailable")
CALL = "invocation:weather:assess"
BINDING = "binding:" + CALL
ROOTS = ("root:outing", "root:weather")
FORECAST_STAGES = ("absent", "commit", "authorize", "reserve", "dispatch", "result", "settle", "select")


class InvariantError(ValueError):
    pass


def require(condition, message):
    if not condition:
        raise InvariantError(message)


@dataclass(frozen=True)
class Config:
    variant: str
    outcomes: tuple
    allowance: int = 1
    domain: str = "broad"
    no_result: tuple = ()

    def __post_init__(self):
        if self.variant not in ("direct", "delegated", "confirm-first"):
            raise ValueError("unsupported variant")
        if type(self.outcomes) is not tuple or len(self.outcomes) > 2 or any(x not in CLASSES for x in self.outcomes):
            raise ValueError("unsupported frozen outcomes; require a tuple of zero through two known classes")
        if type(self.allowance) is not int or self.allowance not in (0, 1):
            raise ValueError("unsupported helper allowance")
        if self.domain not in DOMAINS:
            raise ValueError("unsupported scheduling domain")
        if (type(self.no_result) is not tuple or any(type(i) is not int or i < 0 or i >= len(self.outcomes) for i in self.no_result)
                or tuple(sorted(set(self.no_result))) != self.no_result):
            raise ValueError("unsupported no-result mask")


@dataclass(frozen=True)
class Event:
    """One retained protocol boundary; identities are symbolic, injective originals."""
    stage: str
    operation: str
    actor: str
    target: str = ""
    candidate: int = -1
    binding: str = ""
    disposition: str = ""


@dataclass(frozen=True)
class Message:
    operation: str
    sender: str
    receiver: str
    transport: str
    binding: str = ""
    committed: bool = False
    authorization: str = ""
    source_ended_at_authorization: bool = False
    admission: str = ""
    refusal: str = ""
    acknowledged: bool = False
    selected: bool = False


@dataclass(frozen=True)
class Forecast:
    operation: str
    owner: str
    candidate: int
    stage: int = 0
    result: str = ""


@dataclass(frozen=True)
class Invocation:
    operation: str = CALL
    committed: bool = False
    authorized: bool = False
    admission: str = ""
    member: str = ""
    binding: str = ""
    request_observed: bool = False
    request_selected: bool = False
    frozen: tuple = ()
    published: bool = False
    result_copy: bool = False
    outcome_delivered: bool = False
    selected: bool = False


@dataclass(frozen=True)
class State:
    outing: str
    weather: str
    helper: str
    messages: tuple
    forecasts: tuple
    invocation: Invocation = Invocation()
    assessment: tuple = ()
    decision: tuple = ()
    charges: tuple = (0, 0, 0)  # invocations, created runs, effects
    contacts: int = 0
    root_freezes: tuple = (None, None)
    root_publications: tuple = (False, False)


def initial(config):
    owner = "helper" if config.variant == "delegated" else "weather"
    messages = tuple(Message("message:" + sender + ":" + name, sender, receiver,
                             "transport:" + name, BINDING if name == "reply" else "")
                     for name, sender, receiver in (("proposal", "outing", "weather"),
                                                   ("assessment", "weather", "outing"),
                                                   ("decision", "outing", "weather"),
                                                   ("reply", "helper", "weather")))
    return State("start", "await_proposal", "absent", messages,
                 tuple(Forecast("forecast:" + owner + ":" + str(i), owner, i) for i in range(len(config.outcomes))))


def terminal(position):
    return position in ("succeeded", "failed")


def authorization_disposition(source_ended, reply_binding, original_binding):
    return "allow" if not source_ended or (reply_binding != "" and reply_binding == original_binding) else "deny"


def policy(outcomes):
    for i, outcome in enumerate(outcomes):
        if outcome == "suitable":
            return ("recommended", i)
    return ("unavailable", -1) if "unavailable" in outcomes else ("no_match", -1)


def rows(state):
    return tuple((f.candidate, f.result, f.operation) for f in state.forecasts)


def message_update(state, index, **changes):
    messages = list(state.messages)
    messages[index] = replace(messages[index], **changes)
    return replace(state, messages=tuple(messages))


def forecast_update(state, index, **changes):
    forecasts = list(state.forecasts)
    forecasts[index] = replace(forecasts[index], **changes)
    return replace(state, forecasts=tuple(forecasts))


def event(stage, operation, actor, target="", candidate=-1, binding="", disposition=""):
    return Event(stage, operation, actor, target, candidate, binding, disposition)


def can_end(config, state, actor):
    return config.domain == "broad" or all(not m.committed or m.sender != actor or m.binding or m.authorization == "allow"
                                         for m in state.messages)


def root_intents(state, actor):
    originals = [f.operation for f in state.forecasts if f.owner == actor and f.stage > 0]
    if actor == "weather" and state.invocation.committed:
        originals.append(CALL)
    return tuple(sorted(originals))


def successors(config, state):
    """Yield (Event, State); no hidden source denial, retry, or network-result edge."""
    m = state.messages
    inv = state.invocation
    if state.outing == "start":
        yield event("message.commit", m[0].operation, "outing", "weather"), replace(message_update(state, 0, committed=True), outing="await_assessment")
    if state.outing == "ready_decision":
        yield event("message.commit", m[2].operation, "outing", "weather"), replace(message_update(state, 2, committed=True), outing="ready_end", decision=policy(config.outcomes))
    if state.weather == "work" and config.variant == "delegated":
        yield event("invocation.commit", CALL, "weather", "helper"), replace(state, weather="await_helper", invocation=replace(inv, committed=True))
    if inv.committed and not inv.authorized:
        yield event("invocation.authorize", CALL, "weather", "helper", disposition="allow"), replace(state, invocation=replace(inv, authorized=True))
    if inv.authorized and not inv.admission:
        accepted = config.allowance == 1
        yield event("invocation.admit", CALL, "weather", "helper", binding=BINDING if accepted else "", disposition="accepted" if accepted else "invocation_limit"), replace(
            state, invocation=replace(inv, admission="accepted" if accepted else "refused", member="helper" if accepted else "", binding=BINDING if accepted else "", request_observed=accepted),
            helper="await_request" if accepted else "absent", charges=(1, 1, 0) if accepted else (0, 0, 0))
    if inv.request_observed and not inv.request_selected:
        yield event("invocation.request.select", CALL, "helper", "weather", binding=BINDING), replace(state, invocation=replace(inv, request_selected=True), helper="work")

    owner = "helper" if config.variant == "delegated" else "weather"
    working = getattr(state, owner) == "work" and config.variant != "confirm-first"
    for i, f in enumerate(state.forecasts):
        if f.stage == 0 and working and all(p.stage == 7 for p in state.forecasts[:i]):
            yield event("forecast.commit", f.operation, owner, candidate=i), forecast_update(state, i, stage=1)
        elif 1 <= f.stage < 7:
            next_stage = f.stage + 1
            if next_stage == 5 and i in config.no_result:
                continue
            nxt = forecast_update(state, i, stage=next_stage, result=config.outcomes[i] if next_stage >= 5 else "")
            if next_stage == 3:
                nxt = replace(nxt, charges=(state.charges[0], state.charges[1], state.charges[2] + 1))
            if next_stage == 4:
                nxt = replace(nxt, contacts=state.contacts + 1)
            yield event("forecast." + FORECAST_STAGES[next_stage], f.operation, owner, candidate=i), nxt
    done = all(f.stage == 7 for f in state.forecasts)
    if working and done:
        if owner == "helper":
            yield event("message.commit", m[3].operation, "helper", "weather", binding=BINDING), replace(message_update(state, 3, committed=True), helper="ready_end", assessment=rows(state))
        elif config.variant == "direct":
            yield event("message.commit", m[1].operation, "weather", "outing"), replace(message_update(state, 1, committed=True), weather="await_decision", assessment=rows(state))
    if state.helper == "ready_end":
        frozen = tuple(sorted([m[3].operation] + [f.operation for f in state.forecasts]))
        yield event("helper.end_freeze", CALL, "helper", binding=BINDING, disposition="succeeded"), replace(state, helper="succeeded", invocation=replace(inv, frozen=frozen))
    if inv.admission and not inv.published and (inv.admission == "refused" or (state.helper == "succeeded" and m[3].admission == "accepted")):
        accepted = inv.admission == "accepted"
        yield event("invocation.publish_deliver", CALL, "weather", "weather", binding=inv.binding, disposition="succeeded" if accepted else "rejected"), replace(
            state, invocation=replace(inv, published=True, result_copy=accepted, outcome_delivered=True))
    if inv.outcome_delivered and not inv.selected:
        yield event("invocation.outcome.select", CALL, "weather", binding=inv.binding), replace(state, invocation=replace(inv, selected=True), weather="ready_assessment" if inv.admission == "accepted" else "ready_fail")
    if state.weather == "ready_assessment":
        yield event("message.commit", m[1].operation, "weather", "outing"), replace(message_update(state, 1, committed=True), weather="await_decision")

    for i, msg in enumerate(m):
        if not msg.committed:
            continue
        if not msg.authorization:
            closed = terminal(getattr(state, msg.sender))
            disposition = authorization_disposition(closed, msg.binding, inv.binding)
            yield event("message.authorize", msg.operation, msg.sender, msg.receiver, binding=msg.binding, disposition=disposition), message_update(state, i, authorization=disposition, source_ended_at_authorization=closed)
        if msg.authorization and not msg.admission:
            reason = "source_denied:source_closed" if msg.authorization == "deny" else "target_closed" if not msg.binding and terminal(getattr(state, msg.receiver)) else ""
            disposition = "refused" if reason else "accepted"
            yield event("message.admit", msg.operation, msg.sender, msg.receiver, binding=msg.binding, disposition=reason or disposition), message_update(state, i, admission=disposition, refusal=reason)
        if msg.admission and not msg.acknowledged:
            yield event("message.acknowledge", msg.operation, msg.sender, msg.receiver, binding=msg.binding), message_update(state, i, acknowledged=True)
        if msg.admission == "accepted" and not msg.selected and not msg.binding:
            ready = ((i == 0 and state.weather == "await_proposal") or (i == 1 and state.outing == "await_assessment") or (i == 2 and state.weather == "await_decision"))
            if ready:
                nxt = message_update(state, i, selected=True)
                if i == 0:
                    nxt = replace(nxt, weather="await_decision_first" if config.variant == "confirm-first" else "work")
                elif i == 1:
                    nxt = replace(nxt, outing="ready_decision")
                else:
                    nxt = replace(nxt, weather="ready_end")
                yield event("message.select", msg.operation, msg.receiver, msg.sender), nxt

    for i, actor in enumerate(("outing", "weather")):
        position = getattr(state, actor)
        if position in ("ready_end", "ready_fail") and can_end(config, state, actor):
            status = "failed" if position == "ready_fail" or policy(config.outcomes)[0] == "unavailable" else "succeeded"
            freezes = list(state.root_freezes)
            freezes[i] = root_intents(state, actor)
            yield event("root.end_freeze", ROOTS[i], actor, disposition=status), replace(state, **{actor: status}, root_freezes=tuple(freezes))
        if terminal(position) and not state.root_publications[i] and not root_debts(state, i):
            publications = list(state.root_publications)
            publications[i] = True
            yield event("root.publish", ROOTS[i], actor, disposition=position), replace(state, root_publications=tuple(publications))


def root_debts(state, root):
    """Store descendant/settlement debt, distinct from caller/message obligations."""
    if root == 0:
        return ()
    result = [f.operation for f in state.forecasts if 0 < f.stage < 6]
    inv = state.invocation
    if inv.committed and (not inv.admission or (inv.admission == "accepted" and not inv.published)):
        result.append(CALL)
    return tuple(result)


def debts(config, state):
    result = []
    for msg in state.messages:
        if not msg.committed:
            continue
        if not msg.authorization:
            result.append(msg.operation + ":authorization")
        if not msg.admission:
            result.append(msg.operation + ":admission")
        if not msg.acknowledged:
            result.append(msg.operation + ":acknowledgment")
        if not msg.binding and not msg.selected:
            result.append(msg.operation + ":selection")
        if msg.binding and msg.admission != "accepted":
            result.append(msg.operation + ":recorded_reply")
    inv = state.invocation
    if inv.committed:
        if not inv.admission:
            result.append(CALL + ":admission")
        if not inv.published:
            result.append(CALL + ":publication_delivery")
        if not inv.selected:
            result.append(CALL + ":selection")
        if inv.admission == "accepted" and not inv.request_selected:
            result.append(CALL + ":request_selection")
    for f in state.forecasts:
        if 0 < f.stage < 7:
            result.append(f.operation + ":" + FORECAST_STAGES[f.stage + 1])
    # Unstarted obligations are retained by the frozen candidate list, not invented originals.
    if config.variant != "confirm-first" and inv.admission != "refused":
        result += ["candidate:" + str(f.candidate) + ":forecast" for f in state.forecasts if f.stage == 0]
    if not state.messages[1].selected:
        result.append("outing:assessment_receive")
    if not state.messages[2].selected:
        result.append("weather:decision_receive")
    for i, actor in enumerate(("outing", "weather")):
        if not terminal(getattr(state, actor)):
            result.append(actor + ":local_end")
        if not state.root_publications[i]:
            result.append(ROOTS[i] + ":publication")
    if inv.admission == "accepted" and not terminal(state.helper):
        result.append("helper:local_end")
    return tuple(sorted(result))


def state_view(config, state):
    """Inspectable cut plus separate obligations/costs for raw-reader consumers."""
    validate_state(config, state)
    view = asdict(state)
    view["debts"] = debts(config, state)
    view["root_descendant_debts"] = {ROOTS[i]: root_debts(state, i) for i in range(2)}
    view["candidate_cursor"] = next((f.candidate for f in state.forecasts if f.stage < 7), len(state.forecasts))
    view["charges_by_original"] = ([{"operation": CALL, "invocations": 1, "created_runs": 1, "effects": 0}]
                                   if state.invocation.admission == "accepted" else [])
    view["charges_by_original"] += [{"operation": f.operation, "invocations": 0, "created_runs": 0, "effects": 1}
                                    for f in state.forecasts if f.stage >= 3]
    view["remaining"] = {"helper_invocations": (config.allowance if config.variant == "delegated" else 0) - state.charges[0],
                         "created_runs": int(config.variant == "delegated") - state.charges[1], "effects": len(config.outcomes) - state.charges[2],
                         "in_flight": 1 - sum(3 <= f.stage < 5 for f in state.forecasts)}
    view["depth"] = int(state.invocation.admission == "accepted")
    return view


def completion_allowed(answer, outstanding, roots):
    return bool(answer and answer[0] in ("recommended", "no_match") and not outstanding and all(roots))


def successful_mark(config, state):
    return (state.outing == state.weather == "succeeded" and state.decision == policy(config.outcomes)
            and completion_allowed(state.decision, debts(config, state), state.root_publications))


def diagnostic_terminal(config, state):
    return (state.decision == ("unavailable", -1) and state.outing == state.weather == "failed"
            and not debts(config, state) and all(state.root_publications))


def validate_state(config, state):
    """Independent ledger, prefix, binding, control, and completion constraints."""
    baseline = initial(config)
    require(len(state.root_freezes) == len(state.root_publications) == 2, "root cardinality")
    require(type(state.contacts) is int and all(type(c) is int for c in state.charges), "integer costs")
    require(len(state.messages) == 4 and len(state.forecasts) == len(config.outcomes), "original cardinality")
    require(state.outing in ("start", "await_assessment", "ready_decision", "ready_end", "succeeded", "failed"), "outing control")
    require(state.weather in ("await_proposal", "work", "await_helper", "ready_assessment", "await_decision", "await_decision_first", "ready_end", "ready_fail", "succeeded", "failed"), "weather control")
    require(state.helper in ("absent", "await_request", "work", "ready_end", "succeeded"), "helper control")
    inv = state.invocation
    require(inv.operation == CALL, "original invocation identity")
    require(inv.admission in ("", "accepted", "refused"), "invocation admission class")
    require(not inv.authorized or inv.committed, "invocation authorization before commit")
    require(not inv.admission or inv.authorized, "invocation admission before authorization")
    accepted = inv.admission == "accepted"
    require(not accepted or config.allowance == 1, "invocation allowance exceeded")
    require(inv.admission != "refused" or config.allowance == 0, "false invocation refusal")
    require((inv.member, inv.binding, inv.request_observed) == (("helper", BINDING, True) if accepted else ("", "", False)), "exact original binding/member/request")
    require((state.helper != "absent") == accepted, "member existence")
    require(not inv.request_selected or inv.request_observed, "request selection before admission")
    require(inv.published == inv.outcome_delivered, "atomic invocation publication/delivery")
    require(inv.result_copy == (inv.published and accepted), "accepted result copy only")
    require(not inv.selected or inv.outcome_delivered, "parent selection before publication")
    require(not inv.published or bool(inv.admission), "publication before admission")
    if accepted and inv.published:
        require(state.helper == "succeeded" and state.messages[3].admission == "accepted", "publication before frozen reply admission")
    if state.helper == "succeeded":
        require(inv.frozen == tuple(sorted([state.messages[3].operation] + [f.operation for f in state.forecasts])), "freeze must retain complete original work and reply")
    else:
        require(not inv.frozen, "freeze before helper End")
    if config.variant != "delegated":
        require(inv == Invocation() and state.helper == "absent" and not state.messages[3].committed, "unexpected helper in direct/control")
    if inv.committed:
        require(state.messages[0].selected, "invocation before original proposal selection")
    for i, msg in enumerate(state.messages):
        original = baseline.messages[i]
        require((msg.operation, msg.sender, msg.receiver, msg.transport, msg.binding) ==
                (original.operation, original.sender, original.receiver, original.transport, original.binding), "original message/binding/transport identity")
        require(msg.authorization in ("", "allow", "deny") and msg.admission in ("", "accepted", "refused"), "message stage class")
        require(not msg.authorization or msg.committed, "message authorization before commit")
        require(not msg.source_ended_at_authorization or (msg.authorization and terminal(getattr(state, msg.sender))), "authorization source snapshot")
        if msg.authorization:
            expected = "allow" if not msg.source_ended_at_authorization or (i == 3 and accepted and msg.binding == inv.binding) else "deny"
            require(msg.authorization == expected, "source authorization disposition")
        require(not msg.admission or bool(msg.authorization), "admission before authorization")
        require(msg.admission != "accepted" or msg.authorization == "allow", "delivery after denied source")
        require(not msg.acknowledged or bool(msg.admission), "acknowledgment before admission")
        require(not msg.selected or (msg.admission == "accepted" and i != 3), "selection without ordinary inbox")
        require((msg.admission == "refused") == bool(msg.refusal), "refusal reason")
        if msg.authorization == "deny" and msg.admission:
            require(msg.refusal == "source_denied:source_closed", "immutable denial lost")
        if config.domain == "preclose-authorization" and not msg.binding and msg.committed and terminal(getattr(state, msg.sender)):
            require(msg.authorization == "allow" and not msg.source_ended_at_authorization, "preclose premise")
    for i, f in enumerate(state.forecasts):
        require((f.operation, f.owner, f.candidate) == (baseline.forecasts[i].operation, baseline.forecasts[i].owner, i), "original forecast identity/owner")
        require(type(f.stage) is int and 0 <= f.stage <= 7, "forecast stage")
        require(f.result == (config.outcomes[i] if f.stage >= 5 else ""), "original forecast result")
        require(i not in config.no_result or f.stage < 5, "invented interrupted result")
        if f.stage:
            require(state.messages[0].selected and config.variant != "confirm-first", "forecast before proposal/control cycle")
            require(all(p.stage == 7 for p in state.forecasts[:i]), "candidate order / previous result selection")
            require(config.variant != "delegated" or inv.request_selected, "forecast before helper request selection")
    expected_cost = (int(accepted), int(accepted), sum(f.stage >= 3 for f in state.forecasts))
    require(state.charges == expected_cost, "original ledger debit mismatch")
    require(state.contacts == sum(f.stage >= 4 for f in state.forecasts), "original dispatch/contact count")
    require(sum(3 <= f.stage < 5 for f in state.forecasts) <= 1, "effect capacity one")
    require(state.charges[2] <= len(config.outcomes), "effect bound")
    if state.messages[1].committed or state.messages[3].committed:
        require(all(f.stage == 7 for f in state.forecasts) and state.assessment == rows(state), "assessment omitted original obligation")
    else:
        require(not state.assessment, "assessment before computation")
    if state.messages[1].committed:
        require(state.messages[0].selected, "assessment before proposal")
        require(config.variant != "delegated" or (inv.selected and accepted), "public assessment before original helper selection")
    if state.messages[3].committed:
        require(inv.request_selected, "reply before helper request selection")
    if state.messages[2].committed:
        require(state.messages[1].selected and state.decision == policy(config.outcomes), "decision before assessment / policy mismatch")
    else:
        require(not state.decision, "decision before Send")
    require((state.outing != "start") == state.messages[0].committed, "proposal control")
    require((state.weather != "await_proposal") == state.messages[0].selected, "proposal selection control")
    require((state.outing in ("ready_decision", "ready_end", "succeeded", "failed")) == state.messages[1].selected, "assessment selection control")
    require((state.outing in ("ready_end", "succeeded", "failed")) == state.messages[2].committed, "decision commit control")
    if state.weather in ("ready_end", "succeeded", "failed") and inv.admission != "refused":
        require(state.messages[2].selected, "Weather End before decision selection")
    if state.weather == "work":
        require(not state.messages[1].committed and not inv.committed, "Weather work control")
    if state.weather == "await_helper":
        require(inv.committed and not inv.selected, "Weather invocation await control")
    if state.weather == "ready_assessment":
        require(inv.selected and accepted and not state.messages[1].committed, "Weather accepted response control")
    if state.weather == "await_decision":
        require(state.messages[1].committed and not state.messages[2].selected, "Weather decision await control")
    if state.weather == "await_decision_first":
        require(config.variant == "confirm-first" and not state.messages[1].committed, "confirm-first control")
    if state.weather == "ready_fail" or (state.weather == "failed" and inv.admission == "refused"):
        require(inv.selected and inv.admission == "refused", "refusal outcome selection before failure")
    if accepted:
        expected_helper = "await_request" if not inv.request_selected else "work" if not state.messages[3].committed else "ready_end"
        require(state.helper == expected_helper or (state.helper == "succeeded" and state.messages[3].committed), "helper request/reply control")
    if inv.admission == "refused":
        require(not state.messages[1].committed and not state.assessment and state.charges == (0, 0, 0), "refusal fabricated assessment or child cost")
    for i, actor in enumerate(("outing", "weather")):
        ended = terminal(getattr(state, actor))
        require((state.root_freezes[i] is not None) == ended, "root End/freeze atomicity")
        if ended:
            require(state.root_freezes[i] == root_intents(state, actor), "root frozen original obligations")
            expected_status = "failed" if (actor == "weather" and inv.admission == "refused") or policy(config.outcomes)[0] == "unavailable" else "succeeded"
            require(getattr(state, actor) == expected_status, "local success/diagnostic status")
        require(not state.root_publications[i] or (ended and not root_debts(state, i)), "root publication with descendant debt")


def validate_mark(config, state, marked):
    """Audit a candidate mark independently of a proposed marking function."""
    validate_state(config, state)
    if marked:
        require(state.decision and state.decision[0] in ("recommended", "no_match"), "invalid successful answer")
        require(state.outing == state.weather == "succeeded" and all(state.root_publications), "invalid terminal completion mark")
        require(not debts(config, state), "completion mark hides obligations")


def advance(config, state, selected_event):
    """Strict step for correspondence; unsupported/misbound/out-of-order events reject."""
    matches = [nxt for ev, nxt in successors(config, state) if ev == selected_event]
    if len(matches) != 1:
        raise ValueError("unsupported or disabled event: " + repr(selected_event))
    validate_state(config, matches[0])
    return matches[0]


def validate_transition(config, before, selected_event, after):
    require(advance(config, before, selected_event) == after, "transition omitted or changed retained obligation/cost")


def repeat_service(config, state, selected_event):
    """Same-identity authorization/admission/ack service is durable and idempotent.

    Return the first retained disposition, never recalculate it after closure.
    This helper adds no exploration edge and accepts no alternative transport.
    """
    for msg in state.messages:
        if selected_event.operation != msg.operation:
            continue
        dispositions = {"message.authorize": msg.authorization,
                        "message.admit": msg.refusal or msg.admission,
                        "message.acknowledge": "" if msg.acknowledged else None}
        disposition = dispositions.get(selected_event.stage)
        expected = event(selected_event.stage, msg.operation, msg.sender, msg.receiver, binding=msg.binding, disposition=disposition or "")
        if disposition is not None and (disposition or msg.acknowledged) and selected_event == expected:
            return state
    return advance(config, state, selected_event)


@dataclass
class Graph:
    config: Config
    states: list
    edges: list
    parents: list
    completing: set
    marks: set
    diagnostics: set

    def trace(self, index):
        path = []
        while self.parents[index] is not None:
            index, ev = self.parents[index]
            path.append(asdict(ev))
        return list(reversed(path))


def explore(config):
    start = initial(config)
    states, parents, edges, reverse = [start], [None], [], [[]]
    indexes = {start: 0}
    marks, diagnostics = set(), set()
    cursor = 0
    while cursor < len(states):
        state = states[cursor]
        validate_state(config, state)
        mark = successful_mark(config, state)
        validate_mark(config, state, mark)
        if mark:
            marks.add(cursor)
        if diagnostic_terminal(config, state):
            diagnostics.add(cursor)
        outgoing = []
        for ev, nxt in successors(config, state):
            target = indexes.get(nxt)
            if target is None:
                target = len(states)
                indexes[nxt] = target
                states.append(nxt)
                parents.append((cursor, ev))
                reverse.append([])
            outgoing.append((ev, target))
            reverse[target].append(cursor)
        edges.append(outgoing)
        cursor += 1
    completing = set(marks)
    queue = deque(marks)
    while queue:
        for predecessor in reverse[queue.popleft()]:
            if predecessor not in completing:
                completing.add(predecessor)
                queue.append(predecessor)
    return Graph(config, states, edges, parents, completing, marks, diagnostics)


def graph_summary(graph):
    config = graph.config
    failed = [i for i in range(len(graph.states)) if i not in graph.completing]
    categories = {
        "source_refusal": lambda s: s.messages[2].authorization == "deny",
        "source_refusal_acknowledged": lambda s: s.messages[2].authorization == "deny" and s.messages[2].admission == "refused" and s.messages[2].acknowledged,
        "helper_refusal": lambda s: s.invocation.admission == "refused",
        "unknown_forecast": lambda s: any(f.stage == 4 and f.candidate in config.no_result for f in s.forecasts),
        "confirm_first_cycle": lambda s: s.weather == "await_decision_first" and s.outing == "await_assessment",
    }
    witnesses = {}
    if failed:
        witnesses["nonblocking"] = {"state": failed[0], "trace": graph.trace(failed[0]), "debts": debts(config, graph.states[failed[0]])}
    for name, predicate in categories.items():
        index = next((i for i, state in enumerate(graph.states) if predicate(state)), None)
        if index is not None:
            witnesses[name] = {"state": index, "trace": graph.trace(index), "debts": debts(config, graph.states[index])}
    # Retain the broad paired control with an explicit order, not an inferred timestamp.
    if config.domain == "broad":
        before = next((i for i, s in enumerate(graph.states) if s.outing == "ready_end" and not s.messages[2].authorization), None)
        if before is not None:
            state = graph.states[before]
            trace = graph.trace(before)
            for stage, operation in (("message.authorize", "message:outing:decision"),
                                     ("root.end_freeze", "root:outing"),
                                     ("message.admit", "message:outing:decision")):
                ev, state = next((ev, nxt) for ev, nxt in successors(config, state) if ev.stage == stage and ev.operation == operation)
                trace.append(asdict(ev))
            witnesses["allow_before_end_admit_after_end"] = {"trace": trace, "debts": debts(config, state),
                "has_successful_suffix": state in {graph.states[i] for i in graph.completing},
                "label": "paired ordered control; not claimed shortest"}
    answers = sorted({state.decision for i, state in enumerate(graph.states) if i in graph.marks})
    return {"config": asdict(config), "states": len(graph.states), "edges": sum(map(len, graph.edges)),
            "successful_marks": len(graph.marks), "diagnostic_terminal_states": len(graph.diagnostics),
            "can_complete_states": len(graph.completing), "initial_can_complete": 0 in graph.completing,
            "nonblocking": len(graph.completing) == len(graph.states), "semantic_answers": answers,
            "successful_costs": sorted({graph.states[i].charges for i in graph.marks}),
            "counterexamples": witnesses}


def configurations():
    for domain in DOMAINS:
        for n in range(3):
            for outcomes in itertools.product(CLASSES, repeat=n):
                for mask in itertools.product((False, True), repeat=n):
                    lost = tuple(i for i in range(n) if mask[i])
                    for variant in ("direct", "delegated"):
                        for allowance in (0, 1):
                            yield Config(variant, outcomes, allowance, domain, lost)
        yield Config("confirm-first", ("suitable", "unsuitable"), 1, domain)


def enumerate_models():
    results = [graph_summary(explore(config)) for config in configurations()]
    comparisons = []
    for domain in DOMAINS:
        for n in range(3):
            for outcomes in itertools.product(CLASSES, repeat=n):
                # Compare only adequate admission and known outcomes, preserving diagnostics.
                pair = [next(r for r in results if r["config"] == asdict(Config(variant, outcomes, 1, domain))) for variant in ("direct", "delegated")]
                comparisons.append({"domain": domain, "outcomes": outcomes, "adequate_admission": True,
                                    "known_results": True, "semantic_answer_sets_equal": pair[0]["semantic_answers"] == pair[1]["semantic_answers"],
                                    "successful_replacement": bool(pair[0]["semantic_answers"] and pair[0]["semantic_answers"] == pair[1]["semantic_answers"]),
                                    "direct_costs": pair[0]["successful_costs"], "delegated_costs": pair[1]["successful_costs"]})
    return {"schema": VERSION, "scope": {"candidate_counts": [0, 1, 2], "domains": DOMAINS,
            "classes": CLASSES, "no_result": "Every subset of originals; chosen continuation has no later retained result and no retry. Missing evidence alone does not prove no authentic result can ever arrive.",
            "transport": "One transport identity per original; repeated same-identity service idempotent; distinct transports outside domain.",
            "limits": "Finite protocol only; no concrete Store correspondence, scheduler theorem, exact-debit bisimulation, or continuation-monitor theorem."},
            "totals": {"configurations": len(results), "states": sum(r["states"] for r in results), "edges": sum(r["edges"] for r in results)},
            "results": results, "comparisons": comparisons}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    enumeration = commands.add_parser("enumerate")
    enumeration.add_argument("--output", required=True)
    for command in ("check", "compare"):
        evidence = commands.add_parser(command)
        if command == "check":
            evidence.add_argument("--store", required=True)
            evidence.add_argument("--report")
        else:
            evidence.add_argument("--baseline", required=True)
            evidence.add_argument("--candidate", required=True)
            evidence.add_argument("--replacement")
        for name in ("configuration", "fixture", "manifest", "output"):
            evidence.add_argument("--" + name, required=True)
    args = parser.parse_args(argv)
    if "EXECUTION_PROVENANCE" not in globals():
        parser.error("execute through run.py with isolated Python; unbound source execution is unsupported")
    try:
        if args.command == "enumerate":
            result = enumerate_models()
        elif args.command == "check":
            result = EVIDENCE_API["check_store"](args.store, args.configuration, args.fixture, args.manifest, args.report)
        else:
            result = EVIDENCE_API["compare_stores"](args.baseline, args.candidate, args.configuration, args.fixture, args.manifest, args.replacement)
    except (InvariantError, ValueError, KeyError, sqlite3.Error) as error:
        result = {"accepted": False, "invariant": str(error)}
    result["execution"] = dict(EXECUTION_PROVENANCE, python=platform.python_version(), sqlite=sqlite3.sqlite_version)
    with open(args.output, "w", encoding="utf-8") as target:
        json.dump(result, target, indent=2, sort_keys=True)
        target.write("\n")
    print(json.dumps(result.get("totals", {k: result[k] for k in ("accepted", "classification", "invariant") if k in result}), sort_keys=True))
    if result.get("accepted") is False:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
