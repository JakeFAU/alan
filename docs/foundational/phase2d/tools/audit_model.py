"""Independent finite-protocol audit. No Go imports or calls to the enumerator.

The shared JSON ABI identifies the fields and action names. Legal actions and
their complete replacement state are calculated before consulting outgoing edges.
"""
import argparse
from collections import Counter, deque
import copy
import gzip
import itertools
import json
from pathlib import Path

MODEL = Path(__file__).resolve().parents[1] / "evidence/model"
LAUNCH = "order equal-order cap-one cap-one-granted".split()
READ = ("adopt-duplicate later-error direct-existing retained-failure read-only "
        "absent-refresh late-application loss-refresh exhaust import-kill selection-kill "
        "lost-found intact-found reverse-read competing invalid-role incompatible "
        "manual-ack fresh-ack slow-read").split()
DELAY = {"absent-refresh", "late-application", "loss-refresh"}
MODES = ["answer", "fail", "absent", "pending"]
NAMES = "LR"


def require(ok, reason):
    if not ok:
        raise ValueError(reason)


def manifest():
    def profile(n, section, shape="fork", assignment="AB", modes=("answer", "answer"), rid=2):
        return dict(Name=n, Case=n, Slice=section, Shape=shape, Assignment=assignment,
                    IDs=[1, rid], Modes=list(modes), Deltas=[1, 1 if n == "equal-order" else 2],
                    Queries=[2 if n in {"adopt-duplicate", "later-error", "incompatible"} else 1,
                             2 if n in {"absent-refresh", "late-application"} else 1],
                    Capacity=1 if n in {"cap-one", "cap-one-granted", "slow-read"} else 2)
    result = [profile(n, "launch") for n in LAUNCH] + [profile(n, "read") for n in READ]
    for a, l, r, rid in itertools.product(("AA", "AB", "BA", "BB"), MODES, MODES, (1, 2)):
        result.append(profile(f"fork-{a}-{l}-{r}-id{rid}", "tree", "fork", a, (l, r), rid))
    for shape, r in itertools.product(("relay-fork", "fork-relay"), ("answer", "fail", "pending")):
        result.append(profile(f"{shape}-{r}", "tree", shape, modes=("answer", r)))
    return {p["Name"]: p for p in result}


def tree(p):
    modes = p["Modes"]
    frontier = ("await-left" if modes[0] == "pending" else
                "left-selected/await-right" if modes[1] == "pending" else "complete")
    extra = int(p["Shape"] != "fork")
    tags = dict(zip(MODES, ["answer", "candidate_failed", "candidate_missing_reply", "pending"]))
    return dict(Leaves=[dict(ID=p["IDs"][i], Value=i, Tag=tags[m],
                             Answers=[1-i] if m == "answer" else []) for i, m in enumerate(modes)],
                Outcome="pending" if "pending" in modes else "answer" if modes.count("answer") == 2 else "diagnosed",
                Frontier=frontier, ActualSlots=9+p["Assignment"].count("B")+extra,
                CommonSlots=11+extra, Depth=4+extra)


def receipt(role=0, number=0, total=0):
    return dict(Role=role, Mutation=number, Total=total)


def query(**kw):
    return dict(Phase="", Status="", Adoption="", Ord=0, Serial=0, At=0, Receipt=receipt()) | kw


def entry(i=0, j=0, ordinal=0, at=0, proof=None):
    return dict(Target=i, Query=j, Ordinal=ordinal, At=at, Receipt=proof or receipt())


def side():
    return dict(ReportTarget=0, ReportProof=receipt(), Mut="requested", Original="", Report="", Ack="",
                Closed=False, Q=False, Settled=False, Published=False, AckSelected=False, ResolverEnd=False,
                Root=False, Query=[query(),query()], Witness=0, Proof=receipt(), Conflict=False)


def initial(p,t):
    case = p["Name"]
    costs = [dict.fromkeys("I R Mut Read Ord Post Get Apply".split(),0) for _ in range(4)]
    s = dict(Owner=1, Epoch=1, Losses=0, S=[side(),side()], Extra=query(), World=[receipt(),receipt()],
             Serial=0, Total=0, Log=[entry() for _ in range(5)], Cost=costs, TreeStep=case!="slow-read",
             Joined=0, Decision="", ParentEnd=False, ParentRoot=False)
    for c,n in zip(costs,[1,1,t["ActualSlots"]-2,0]):
        c.update(I=n,R=n)
    if p["Slice"]=="launch":
        if p["Capacity"]==1:
            s["S"][0]["Mut"]="admitted"
            costs[0]["Mut"]=1
            if case=="cap-one-granted":
                s["S"][1]["Mut"]="granted"
        return s
    order = [0] if case in DELAY else [1,0] if case=="reverse-read" else [0,1]
    for number,i in enumerate(order,1):
        s["Total"] += p["Deltas"][i]
        s["World"][i]=receipt(i+1,number,s["Total"])
        costs[i]["Apply"]=1
    for i,v in enumerate(s["S"]):
        v.update(Mut="handled",Closed=True,Q=True)
        costs[i].update(Mut=1,Post=1)
        if i==0 and case in {"direct-existing","retained-failure"}:
            v.update(Original="direct" if case=="direct-existing" else "failure",Settled=True,Published=True)
        if p["Slice"]=="tree":
            proof=s["World"][i]
            v.update(Original="imported",Settled=True,Published=True,Report="booked",ReportTarget=i+1,
                     ReportProof=proof,Witness=1,Proof=proof)
            v["Query"][0]=query(Ord=1,Serial=i+1,At=2,Receipt=proof,Phase="selected",Status="found",Adoption="adopted")
            s["Serial"]+=1
            s["Log"][i]=entry(i+1,1,1,2,proof)
            costs[i].update(Read=1,Ord=1,Get=1)
    return s


def actions(p,t,s):
    """Yield label and a path-to-replacement map; all unmentioned fields are immutable."""
    result={}
    case=p["Name"]
    def emit(name, edits):
        require(name not in result,("ambiguous predicted action",name))
        result[name]=edits
    def spend(i,k):
        return {("Cost",i,k):s["Cost"][i][k]+1}
    def app(i):
        total=s["Total"]+p["Deltas"][i]
        number=1+sum(r["Role"]!=0 for r in s["World"])
        return {("Total",):total,("World",i):receipt(i+1,number,total)}|spend(i,"Apply")
    def loss():
        return {("Owner",):0,("Losses",):s["Losses"]+1}
    def fields(path,**kw):
        return {path+(k,):v for k,v in kw.items()}
    active={"admitted","called","packet","applied"}
    if p["Slice"]=="launch":
        free=sum(v["Mut"] in active for v in s["S"])<p["Capacity"]
        for i,v in enumerate(s["S"]):
            path=("S",i)
            prefix=NAMES[i]+"."
            phase=v["Mut"]
            def mut(action,stage,extra=None,**kw):
                emit(prefix+action,fields(path,Mut=stage,**kw)|(extra or {}))
            if not v["Closed"] and (p["Capacity"]==1 or phase in active|{"handled"}):
                emit(prefix+"end-after-handoff",fields(path,Closed=True))
            if v["Closed"] and not v["Q"] and phase in ("denied","handled"):
                emit(prefix+"Q",fields(path,Q=True))
            if phase=="requested" and free:
                if v["Closed"]:
                    mut("deny-first-permission","first-denied")
                else:
                    mut("allow-and-reserve","reserved",spend(i,"Mut"))
            if phase=="granted" and free:
                mut("reserve-historical-grant","reserved",spend(i,"Mut"))
            if phase=="reserved" and free:
                if v["Closed"]:
                    mut("refuse-current-gate","denied")
                else:
                    mut("admit","admitted")
            if phase=="first-denied":
                mut("record-source-denied","recorded-denial",Original="source-denied")
            if phase=="recorded-denial":
                mut("settle-source-denied","denied",Settled=True)
            for old,new,action,cost in [("admitted","called","enter",{}),("called","packet","ingress",spend(i,"Post")),
                                        ("packet","applied","apply",app(i)),("applied","handled","handle-no-result",{})]:
                if phase==old:
                    mut(action,new,cost)
        return result
    left,right=s["S"]
    if case in DELAY and s["World"][1]["Role"]==0:
        release = (s["Losses"]>0 if case=="loss-refresh" else
                   right["Query"][int(case=="late-application")]["Phase"]=="selected")
        if release:
            emit("R.apply-held-packet",app(1))
    if not s["Owner"]:
        emit("owner.open",{("Owner",):1,("Epoch",):s["Epoch"]+1})
        return result
    if case=="slow-read" and not s["TreeStep"] and left["Query"][0]["Phase"]=="spent":
        emit("pure.private-commit",{("TreeStep",):True})
    def snap(i,j,q,path):
        number=s["Serial"]+1
        apps=sum(x["Role"]!=0 for x in s["World"])
        proof=s["World"][i]
        return (fields(path,Phase="snapshot",Status="found" if proof["Role"] else "not_found",
                       Serial=number,At=apps,Receipt=proof) |
                {("Serial",):number,("Log",number-1):entry(i+1,j+1,q["Ord"],apps,proof)})
    def query_steps(i,j,extra=False):
        v=s["S"][i]
        path=("Extra",) if extra else ("S",i,"Query",j)
        q=s["Extra"] if extra else v["Query"][j]
        prefix="X." if extra else f"{NAMES[i]}.q{j+1}."
        phase=q["Phase"]
        lane=3 if extra else i
        def move(action,new,edits=None,**kw):
            emit(prefix+action,fields(path,Phase=new,**kw)|(edits or {}))
        if phase=="":
            if not extra:
                move("request","issued")
            elif left["Query"][0]["Phase"]=="spent":
                move("request-and-reserve","reserved",spend(lane,"Read"))
        elif phase=="issued":
            move("allow-and-reserve","reserved",spend(lane,"Read"))
        elif phase=="reserved":
            if q["Ord"]>=2:
                move("exhaust","retained",Status="read_exhausted")
            elif v["Original"] and not v["Settled"] or not v["Q"] or not v["Closed"]:
                move("refuse","retained",Status="original_unsettled")
            else:
                move("spend","spent",spend(lane,"Ord"),Ord=q["Ord"]+1)
        elif phase=="spent":
            if i==1 and case=="exhaust":
                move("kill-before-contact","reserved",loss())
            else:
                waiting=(case=="slow-read" and not s["TreeStep"] or
                         case=="competing" and i==0 and (not v["Original"] if extra else s["Extra"]["Phase"]!="spent") or
                         i==1 and not s["World"][1]["Role"] and (case=="absent-refresh" and j==1 or
                                                               case=="loss-refresh" and q["Ord"]==2))
                if not waiting:
                    move("lookup-ingress","contact",spend(lane,"Get"))
        elif phase=="contact":
            emit(prefix+"snapshot",snap(i,j,q,path))
        elif phase=="snapshot":
            if i==1 and q["Ord"]==1 and case in ("loss-refresh","lost-found"):
                emit(prefix+"kill-before-retain",{path:query(Ord=q["Ord"],Phase="reserved")}|loss())
            elif case=="later-error" and i==0 and j==1:
                emit(prefix+"retain-transport-error",{path:query(Phase="retained",Ord=q["Ord"],Status="transport_error")})
            else:
                decision=("none" if not q["Receipt"]["Role"] else "denied" if i==1 and case=="read-only"
                          else {"":"adopted","imported":"duplicate","direct":"existing","failure":"existing"}[v["Original"]])
                extra_edits={("S",i,"Original"):"imported"} if decision=="adopted" else {}
                move("retain" if extra else "retain-and-assess","retained",extra_edits,Adoption=decision)
        elif phase=="retained":
            if i==0 and case=="import-kill" and not s["Losses"]:
                emit(prefix+"kill-after-import",loss())
            elif q["Adoption"]=="adopted" and not v["Settled"]:
                emit(prefix+"settle-original",{("S",i,"Settled"):True})
            else:
                move("settle-query","settled")
        elif phase=="settled":
            edits={}
            proof=q["Receipt"]
            if case=="incompatible" and i==0 and j==1:
                proof=proof|dict(Total=proof["Total"]+1)
                edits[path+("Receipt",)]=proof
            if not extra and q["Status"]=="found":
                if not v["Witness"]:
                    edits.update(fields(("S",i),Witness=j+1,Proof=proof))
                elif proof!=v["Proof"]:
                    edits[("S",i,"Conflict")]=True
            move("select","selected",edits)
    if p["Slice"]=="read":
        for i,v in enumerate(s["S"]):
            if i==1 and left["Report"]!="booked":
                continue
            path=("S",i)
            if v["Original"] and not v["Published"]:
                emit(NAMES[i]+".publish-original",fields(path,Published=True))
            for j in range(p["Queries"][i]):
                if j==0 or v["Query"][j-1]["Phase"]=="selected":
                    query_steps(i,j)
            if not v["Report"] and v["Query"][p["Queries"][i]-1]["Phase"]=="selected":
                target=0 if i==1 and case=="invalid-role" else i
                emit(NAMES[i]+".send-report",fields(path,Report="sent",ReportTarget=target+1,
                                                   ReportProof=s["S"][target]["Proof"]))
            for old,new,label in [("sent","allowed","authorize-report"),("allowed","delivered","admit-report"),
                                  ("delivered","booked","book-report")]:
                if v["Report"]==old:
                    emit(NAMES[i]+"."+label,fields(path,Report=new))
        if case=="competing":
            query_steps(0,2,True)
    if any(v["Report"]!="booked" or v["Original"] and not v["Published"] for v in s["S"]):
        return result
    if case=="competing" and s["Extra"]["Phase"]!="selected":
        return result
    j=s["Joined"]
    if j==1 and case=="selection-kill" and not s["Losses"]:
        emit("owner.kill-after-selection",loss())
    elif j<5:
        choices=[(left["Published"],"L.parent-select-original"),(right["Published"],"R.parent-select-original"),
                 (t["Outcome"]!="pending" and s["TreeStep"],"P.select-tree"),
                 (left["Report"]=="booked","L.parent-select-report"),(right["Report"]=="booked","R.parent-select-report")]
        enabled,label=choices[j]
        if enabled:
            emit(label,{("Joined",):j+1})
    elif not s["Decision"]:
        valid=all(v["Witness"] and not v["Conflict"] and v["ReportTarget"]==i+1 and
                  v["ReportProof"]==s["World"][i] and v["Query"][v["Witness"]-1]["Receipt"]==v["ReportProof"] and
                  v["Query"][v["Witness"]-1]["Adoption"] in {"existing","adopted","duplicate"}
                  for i,v in enumerate(s["S"]))
        emit("P.decide",{("Decision",):("invalid_evidence" if not valid else
                                        "applied" if t["Outcome"]=="answer" else "applied_tree_diagnosed")})
    elif case=="fresh-ack" and left["Ack"]=="sent" and not s["Losses"]:
        emit("owner.kill-before-ack-service",loss())
    else:
        manual=case=="manual-ack"
        if not manual:
            ready_publications={}
            if s["ParentEnd"] and not s["ParentRoot"]:
                ready_publications["P.publish-root"]={("ParentRoot",):True}
            for i,v in enumerate(s["S"]):
                if v["ResolverEnd"] and not v["Root"]:
                    ready_publications[NAMES[i]+".publish-root"]=fields(("S",i),Root=True)
            require(len(ready_publications)<=1,"two unpublished roots in an automatic single-commit turn")
            if ready_publications:
                return ready_publications
        ordinary=manual or all(v["Ack"] in ("","booked") for v in s["S"])
        for i,v in enumerate(s["S"]):
            path=("S",i)
            prefix=NAMES[i]+"."
            phase=v["Ack"]
            if phase=="" and not s["ParentEnd"] and ordinary and (i==0 or left["Ack"]):
                emit(prefix+"send-ack",fields(path,Ack="sent"))
            if phase=="sent":
                allowed=not s["ParentEnd"]
                emit(prefix+("authorize-ack" if allowed else "deny-ack"),fields(path,Ack="allowed" if allowed else "denied"))
            for old,new,label in [("allowed","delivered","admit-ack"),("delivered","booked","book-ack"),
                                  ("denied","refused","refuse-ack"),("refused","closed","book-refusal")]:
                if phase==old:
                    emit(prefix+label,fields(path,Ack=new))
            if phase=="booked":
                if not v["AckSelected"] and ordinary:
                    emit(prefix+"select-ack",fields(path,AckSelected=True))
                elif v["AckSelected"] and not v["ResolverEnd"] and ordinary:
                    emit(prefix+"resolver-end",fields(path,ResolverEnd=True))
                elif v["ResolverEnd"] and not v["Root"]:
                    emit(prefix+"publish-root",fields(path,Root=True))
        if ordinary and right["Ack"] and not s["ParentEnd"]:
            emit("P.end",{("ParentEnd",):True})
        if s["ParentEnd"] and not s["ParentRoot"]:
            emit("P.publish-root",{("ParentRoot",):True})
    return result


def classification(p,t,s):
    l,r=s["S"]
    if s["ParentRoot"]:
        if l["Root"] and r["Root"]:
            return ("designated-roots/"+s["Decision"]+"/extra-reader-selected" if p["Case"]=="competing"
                    else "all-roots/"+s["Decision"])
        if any(v["Ack"]=="closed" for v in (l,r)):
            return "parent-published/ack-refused"
    if p["Slice"]=="launch" and l["Q"] and r["Q"]:
        return "both-originals-quiescent"
    if all(v["Report"]=="booked" for v in (l,r)):
        if s["Joined"]==1 and not r["Original"]:
            return "pending/original-R"
        if s["Joined"]==2 and t["Outcome"]=="pending":
            return "pending/tree/"+t["Frontier"]
    return "prefix"


def key(v):
    return json.dumps(v,sort_keys=True,separators=(",",":"))


def census_and_ranks(rows):
    # Independent breadth-first distances and longest productive suffix ranks.
    todo=deque([0])
    distances={0:0}
    while todo:
        i=todo.popleft()
        for e in rows[i]["Edges"]:
            j=e["To"]
            if j not in distances:
                distances[j]=distances[i]+1
                todo.append(j)
    require(len(distances)==len(rows),"unreachable state")
    ranks={}
    active=set()
    def rank(i):
        if i not in ranks:
            require(i not in active,"nondecreasing productive cycle")
            active.add(i)
            ranks[i]=max((1+rank(e["To"]) for e in rows[i]["Edges"]),default=0)
            active.remove(i)
        return ranks[i]
    rank(0)
    require(all(ranks[i]>ranks[e["To"]] for i,row in enumerate(rows) for e in row["Edges"]),
            "productive edge without decreasing suffix rank")
    return distances,ranks


def check_state(s):
    c=s["Cost"]
    require(s["Owner"] in (0,1) and s["Epoch"]==1+s["Losses"]-(1-s["Owner"]),"owner conservation")
    business=sorted((w for w in s["World"] if w["Role"]),key=lambda w:w["Mutation"])
    require([w["Mutation"] for w in business]==list(range(1,len(business)+1)),"nonserial business order")
    require(all(c[i]["Apply"]==bool(s["World"][i]["Role"]) and c[i]["Apply"]<=c[i]["Post"]<=c[i]["Mut"]<=1
                for i in (0,1)),"business debit/contact conservation")
    require(s["Serial"]==sum(bool(x["Target"]) for x in s["Log"]),"missing endpoint snapshot")
    for i,queries in [(0,s["S"][0]["Query"]),(1,s["S"][1]["Query"]),(3,[s["Extra"]])]:
        require(sum(q["Ord"] for q in queries)==c[i]["Ord"] and
                c[i]["Get"]<=c[i]["Ord"]<=2*c[i]["Read"] and all(q["Ord"] in (0,1,2) for q in queries),
                "physical allowance conservation")
    for v in s["S"]:
        require(not v["Root"] or v["ResolverEnd"] and v["AckSelected"],"terminated without selected own ACK")
        require(not v["Published"] or bool(v["Original"]),"publication without original")
        require(not any(q["Adoption"]=="adopted" and q["Phase"] in ("settled","selected") for q in v["Query"])
                or v["Settled"],"importer selected before original settlement")
    for event in s["Log"][:s["Serial"]]:
        w=s["World"][event["Target"]-1]
        expect=w if w["Mutation"] and w["Mutation"]<=event["At"] else receipt()
        require(event["Receipt"]==expect,"snapshot inconsistent with original prefix")


def audit(g,p,red=()):
    require(g["Profile"]==p,"profile hypotheses")
    t=tree(p)
    require(g["Tree"]==t,"pure occurrence/frontier")
    rows=g["Rows"]
    require(rows[0]["State"]==initial(p,t),"incorrect initial state")
    seen=set()
    sinks=Counter()
    for i,row in enumerate(rows):
        s=row["State"]
        z=key(s)
        require(row["ID"]==i and z not in seen,("duplicate state or vertex ID",i))
        seen.add(z)
        check_state(s)
        expected=actions(p,t,s)
        labels=[edge["Action"] for edge in row["Edges"]]
        require(len(set(labels))==len(labels) and set(labels)==set(expected),
                ("outgoing",p["Name"],i,sorted(set(expected)-set(labels)),sorted(set(labels)-set(expected))))
        for edge in row["Edges"]:
            k=edge["To"];label=edge["Action"]
            require(type(k) is int and 0<=k<len(rows),("bad target",i,label,k))
            n=copy.deepcopy(s)
            for path,value in expected[label].items():
                node=n
                for k0 in path[:-1]:
                    node=node[k0]
                node[path[-1]]=value
            # Strict JSON comparison avoids Python's True == 1 scalar equality.
            require(key(n)==key(rows[k]["State"]),("transition delta",p["Name"],i,label,k))
        cl=classification(p,t,s)
        require(row["Class"]==cl,("frontier",p["Name"],i,cl,row["Class"]))
        if not labels:
            require(cl!="prefix","unfinished unclassified maximum")
            sinks[cl]+=1
    distances,ranks=census_and_ranks(rows)
    for witness in red:
        target=0
        for label in witness["Actions"]:
            target=next((edge["To"] for edge in rows[target]["Edges"] if edge["Action"]==label),None)
            require(target is not None,"RED uses disabled action")
        require(target==witness["End"],"RED ends at a different vertex")
        prop=refuted(p["Case"])
        eligible=[distances[i] for i,row in enumerate(rows) if prop(row["State"])]
        require(prop(rows[target]["State"]) and len(witness["Actions"])==min(eligible),"RED predicate/shortest distance")
    stats=dict(Name=p["Name"],States=len(rows),Edges=sum(len(row["Edges"]) for row in rows),
               Sinks=sum(sinks.values()),Frontiers=dict(sinks))
    return stats,ranks[0]


def refuted(name):
    def predicate(s):
        l,r=s["S"]
        if name=="order":
            return all(w["Role"] for w in s["World"]) and any(w["Mutation"]!=1 or w["Total"]!=1 for w in s["World"])
        if name.startswith("cap-one"):
            return r["Closed"] and r["Mut"]=="denied" and s["Cost"][1]["Post"]==0
        if name=="adopt-duplicate":
            return s["Decision"]=="applied" and l["Query"][1]["Adoption"]=="duplicate"
        if name=="read-only":
            return s["Joined"]==1 and r["Witness"] and not r["Published"]
        if name=="late-application":
            return s["Joined"]==1 and s["World"][1]["Role"] and r["Query"][1]["Status"]=="not_found"
        if name=="lost-found":
            c=s["Cost"][1]
            return s["Decision"]=="applied" and (c["Read"],c["Ord"],c["Get"])==(1,2,2)
        if name=="competing":
            return (s["Extra"]["Phase"]=="selected" and s["Extra"]["Adoption"]=="duplicate"
                    and l["Published"] and not l["Settled"])
        if name=="manual-ack":
            return s["ParentRoot"] and any(v["Ack"]=="closed" for v in s["S"])
        if name=="fork-AB-answer-pending-id1":
            return s["Joined"]==2 and l["Query"][0]["Adoption"]=="adopted" and l["Witness"]==1 and not s["Decision"]
        raise ValueError("unrecognized RED family")
    return predicate


def mutations(g,p):
    """Attack both adjacency and state semantics. Each attack starts from the original export."""
    result={}
    def attack(label,change):
        bad=copy.deepcopy(g)
        change(bad["Rows"])
        try:
            audit(bad,p)
        except ValueError as exc:
            result[label]=str(exc)
        else:
            raise ValueError("undetected mutant: "+label)
    def find_edge(rows,action):
        for row in rows:
            for e in row["Edges"]:
                if e["Action"]==action:
                    return row,e
        raise ValueError("no such test action "+action)
    def after(rows,act):
        _,e=find_edge(rows,act)
        return rows[e["To"]]["State"]
    if p["Name"]=="order":
        attack("omit-enabled",lambda rows:rows[0]["Edges"].pop())
        attack("redirect-edge",lambda rows:rows[0]["Edges"][0].update(To=0))
        attack("duplicate-label",lambda rows:rows[0]["Edges"].append(rows[0]["Edges"][0]))
        attack("missing-application-charge",lambda rows:after(rows,"L.apply")["Cost"][0].update(Apply=0))
        attack("role-swap",lambda rows:after(rows,"L.apply")["World"][0].update(Role=2))
        attack("wrong-total",lambda rows:after(rows,"L.apply")["World"][0].update(Total=999))
    if p["Name"]=="lost-found":
        attack("refund-on-open",lambda rows:after(rows,"owner.open")["Cost"][1].update(Ord=0))
        attack("erase-lost-snapshot",lambda rows:after(rows,"R.q1.kill-before-retain").update(Serial=1))
    if p["Name"]=="read-only":
        attack("denied-adoption-invented",lambda rows:after(rows,"R.q1.retain-and-assess")["S"][1].update(Original="imported"))
    if p["Name"]=="intact-found":
        def add_invalid_ack(rows):
            r=next(r for r in rows if r["State"]["S"][1]["Ack"]=="sent" and not r["State"]["S"][0]["AckSelected"])
            r["Edges"].append(dict(Action="L.select-ack",To=r["ID"]))
        attack("ordinary-selection-outruns-sole-ACK",add_invalid_ack)
        attack("premature-publication",lambda rows:after(rows,"P.decide").update(ParentRoot=True))
    if p["Name"]=="competing":
        attack("duplicate-repays-importer",lambda rows:after(rows,"X.select")["S"][0].update(Settled=True))
    if p["Name"]=="fork-AB-answer-pending-id1":
        attack("pending-called-complete",lambda rows:rows[-1].update(Class="all-roots/applied"))
    return result


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory",type=Path,nargs="?",default=MODEL)
    args=parser.parse_args()
    expected=manifest()
    summary=json.loads((args.directory/"summary.json").read_bytes())
    reds=json.loads((args.directory/"reds.json").read_bytes())
    red_names={"order","cap-one","cap-one-granted","adopt-duplicate","read-only","late-application",
               "lost-found","competing","manual-ack","fork-AB-answer-pending-id1"}
    require(len(reds)==len(red_names) and {a["Profile"] for a in reds}==red_names,"incomplete RED populations")
    stats=[];ranks={};attacks={}
    with gzip.open(args.directory/"graphs.jsonl.gz") as f:
        for raw in f:
            g=json.loads(raw)
            name=g["Profile"]["Name"]
            require(name in expected and name not in ranks,("unexpected/duplicate profile",name))
            data,rank=audit(g,expected[name],[r for r in reds if r["Profile"]==name])
            stats.append(data)
            ranks[name]=rank
            attacks.update(mutations(g,expected[name]))
    require(set(ranks)==set(expected) and stats==summary,"profile population or summary")
    print(json.dumps(dict(profiles=len(stats),states=sum(x["States"] for x in stats),edges=sum(x["Edges"] for x in stats),
                          sinks=sum(x["Sinks"] for x in stats),shortest_RED=len(reds),mutants=attacks,
                          rank_by_profile=ranks),sort_keys=True,indent=2))


if __name__=="__main__":
    main()
