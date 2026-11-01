"""Qualify the public two-effect frame on the frozen Series 1 product matrix."""
import argparse
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import asdict, dataclass
import gzip
import itertools
import json
from pathlib import Path
import runpy
import tempfile

from process_cases import advance, original_loss
from process_lab import Lab, rowset, sha, wait_for, write

ROOT = Path(__file__).resolve().parents[4]
SERIES = runpy.run_path(str(ROOT/"docs/foundational/phase1t/evidence/audit_runtime.py"))
MODES = ("answer", "fail", "absent", "pending")


@dataclass(frozen=True)
class Case:
    assignment: str
    left: str
    right: str
    relay: str = ""
    deficit: str = ""
    recover: bool = False
    slow: bool = False

    @property
    def name(self):
        return "-".join(v for v in (self.assignment,self.left,self.right,self.relay,self.deficit,"slow" if self.slow else "") if v)

    @property
    def term(self):
        return {"":(0,1), "above":((0,1),), "left":((0,),1), "right":(0,(1,))}[self.relay]

    def requests(self):
        return [dict(id=1, value=0, mode=self.left),
                dict(id=1 if self.assignment in ("AA","AB") else 2, value=1, mode=self.right)]


def cases():
    items=[]
    for a,b,m,n in itertools.product("AB","AB",MODES,MODES):
        reconstructed=(a+b,m,n) in {("AB","answer","answer"),("BA","pending","answer")}
        items.append(Case(a+b,m,n,recover=reconstructed))
    items.extend([Case("AB","answer","answer","above"),Case("BA","fail","answer","left"),
                  Case("BB","answer","pending","right")])
    items.extend(Case("BB","answer","answer",deficit=name) for name in ("invocations","runs","depth","effects"))
    items.append(Case("BB","answer","answer","above",recover=True,slow=True))
    return items


class Projection:
    """Present the actual nested T to the historical Series 1 occurrence walk."""
    def __init__(self, db):
        self.db=db
        self.runs={s:{"program_id":i,"projection":p} for s,i,p in rowset(db,"SELECT subject,program_id,projection FROM runs")}
        self.programs={i:json.loads(b) for i,b in rowset(db,"SELECT id,bytes FROM programs")}
        self.it=self.descriptors("intents","id")
        self.ad=self.descriptors("composition_admission_decisions","operation_id")
        self.pub=self.descriptors("composition_publications","operation_id")
        self.children=defaultdict(dict)
        for op in self.ad:
            intent=self.it[op]
            assert intent["Obligation"] not in self.children[intent["Run"]["Subject"]]
            self.children[intent["Run"]["Subject"]][intent["Obligation"]]=op
        # Join actual selected observation and its consuming history. One alone
        # would not establish that an original caller committed that selection.
        self.selections=dict(rowset(db,"""SELECT d.operation_id,h.sequence FROM composition_outcome_deliveries d
            JOIN observations o ON o.id=d.observation_id JOIN history h ON h.selected_observation=o.id
            AND h.subject=o.subject AND h.generation=o.generation AND h.sequence=o.consumed_sequence"""))

    def descriptors(self, table, key):
        return {i:json.loads(buf) for i,buf in rowset(self.db,"SELECT "+key+",descriptor FROM "+table)}

    def child(self, op):
        return self.ad[op]["Members"][0]["Subject"]

    def program(self, who):
        return self.programs[self.runs[who]["program_id"]]

    def projection(self, who):
        return json.loads(self.runs[who]["projection"])

    def selected(self, op):
        return self.selections.get(op)

    def register(self, who):
        return SERIES["record"](self.projection(who)["Data"])

    def message(self, who, obligation):
        observations=rowset(self.db,"""SELECT o.descriptor,o.consumed_sequence,h.sequence FROM observations o
          LEFT JOIN history h ON h.selected_observation=o.id AND h.subject=o.subject AND h.generation=o.generation
          WHERE o.subject=? AND json_extract(o.descriptor,'$.Obligation')=? AND json_extract(o.descriptor,'$.Kind')='message'""",(who,obligation))
        result=[]
        for raw,consumed,seq in observations:
            assert consumed==seq
            result.append((json.loads(raw),seq))
        return result

    def resources(self):
        roots=self.descriptors("composition_roots","id")
        charges=[json.loads(r[0]) for r in rowset(self.db,"SELECT descriptor FROM composition_charges ORDER BY decision_order")]
        return {r["Run"]["Subject"]:{"limits":r["Limits"],"spend":{
            key:sum(c[key] for c in charges if c["RootID"]==rid) for key in ("Invocations","CreatedRuns","Effects")
        }} for rid,r in roots.items()}


def selected_queries(db, who):
    return [json.loads(raw) for raw, in rowset(db,"""SELECT o.descriptor FROM observations o
      JOIN history h ON h.selected_observation=o.id AND h.subject=o.subject
      AND h.generation=o.generation AND h.sequence=o.consumed_sequence
      WHERE o.subject=? AND json_extract(o.descriptor,'$.Obligation')='lookup'""",(who,))]


def grade_limit(case, s, receipts, resources):
    desc=[(op,v["Reason"]) for op,v in s.ad.items() if not v["Accepted"]]
    attempts=s.descriptors("adapter_attempts","intent_id")
    excluded=[(op,v["Reason"]) for op,v in attempts.items() if not v["Allowed"]]
    reason={"invocations":"invocation_limit","runs":"created_run_limit","depth":"depth_limit","effects":"effect_limit"}[case.deficit]
    if case.deficit=="effects":
        assert len(receipts)==1 and len(excluded)==1 and excluded[0][1]==reason,(receipts,excluded)
        assert resources["parent"]["spend"]["Effects"]==1
    else:
        assert len(receipts)==2 and desc and {r for _,r in desc}=={reason},(desc,reason)
        for op,_ in desc:
            assert not rowset(s.db,"SELECT 1 FROM composition_charges WHERE operation_id=? AND kind='invocation'",(op,))
        used=resources["parent"]["spend"]
        assert used["Invocations"]==used["CreatedRuns"]==(9 if case.deficit=="depth" else 10),used
    return {"refusals":desc,"effect_refusals":excluded,"decision":s.register("parent").get("decision")}


def grade(case, lab):
    s=Projection(lab.db)
    p=s.register("parent")
    root=s.resources()
    world=[json.loads(r[0]) for r in rowset(lab.world,"SELECT receipt FROM applications ORDER BY number")]
    if case.deficit:
        result=grade_limit(case,s,world,root)
    else:
        pending="pending" in (case.left,case.right)
        expected="pending" if pending else "applied" if case.left==case.right=="answer" else "applied_tree_diagnosed"
        reqs=case.requests()
        tree=p["tree_invocation"]
        SERIES["walk"](s,tree,case.term,case.assignment,reqs,not pending)
        # Each A leaf spends three invocation/run slots, each B four. Interior
        # relay/fork entries and the two issuing children are separate debits.
        base=2+1+int(bool(case.relay))
        used=base+sum(3 if a=="A" else 4 for a in case.assignment)
        assert root["parent"]["spend"]=={"Invocations":used,"CreatedRuns":used,"Effects":2}
        bits={name:s.selected(p[name+"_invocation"]) for name in ("l","r")}
        assert 0<bits["l"]<bits["r"]
        ids={}
        seqs={}
        receipts={json.loads(w["Key"])[2]:w for w in world}
        assert len(receipts)==len(world)==2
        for role,delta in (("l",1),("r",2)):
            target=p[role+"_target"]
            who="resolver_"+role
            res=s.register(who)
            receipt=receipts[target]
            ids[role]=target
            assert receipt["Delta"]==delta and receipt["Mutation"] in (1,2)
            assert receipt["Total"]==(delta if receipt["Mutation"]==1 else 3)
            assert res["witness"] and res["witness_target"]==target and res["witness_query"]==res["query1"]
            assert (res["witness_mutation"],res["witness_total"])==(receipt["Mutation"],receipt["Total"])
            assert res["query1_adoption"]==("adopted" if case.recover else "existing")
            q=selected_queries(lab.db,who)
            assert len(q)==1 and q[0]["Operation"]==res["query1"],(q,res["query1"])
            assert root[who]["spend"]=={"Effects":1,"Invocations":0,"CreatedRuns":0}
            report=s.message("parent",role+"_report")
            assert len(report)==1 and report[0][0]["Source"]==who and report[0][0]["Payload"]["target"]["Text"]==target
            seqs[role]=report[0][1]
            acks=s.message(who,role+"_report")
            if pending:
                assert seqs[role] is None and not acks and s.projection(who)["Status"]=="open"
            else:
                assert len(acks)==1 and acks[0][1] and acks[0][0]["Source"]=="parent"
                assert acks[0][0]["Payload"]["target"]["Text"]==target and s.projection(who)["Status"]=="succeeded"
                assert p[role+"_class"]==("applied_issued" if case.recover else "applied_direct")
        if pending:
            assert s.projection("parent")["Status"]=="open" and "decision" not in p
            SERIES["waiting"](s,"parent",tree)
        else:
            assert bits["r"]<s.selected(tree)<seqs["l"]<seqs["r"]
            assert p["decision"]==expected and s.projection("parent")["Status"]==("succeeded" if expected=="applied" else "failed")
        orders=rowset(lab.db,"SELECT count(*) FROM receipt_read_ordinals")[0][0]
        reads=rowset(lab.world,"SELECT count(*) FROM receipt_reads")[0][0]
        assert orders==reads==2
        result={"decision":expected,"originals":ids,"selected_originals":bits,"tree_selection":s.selected(tree),"report_selections":seqs,"read_opportunities":orders}
    return {"case":asdict(case),"resources":root,"world":world,**result}


def slow_read(lab):
    # The public empty input releases only L. With the original workers gone,
    # one fresh physical slot suffices. Pause real T instructions at staging
    # until the read has a retained ordinal and its worker can reach the relay.
    lab.service("l")
    s=Projection(lab.db)
    tree=s.register("parent")["tree_invocation"]
    member=s.child(tree)
    assert s.projection(member)["Status"]=="open" and tree not in s.pub
    lab.relay.held=True
    worker=lab.owner("execution.staged",subject=member,capacity=1)
    # Two old original packets passed through this same scheduling relay.
    # original_loss consumed their ingress notifications before the executor died.
    for _ in range(200):
        event=lab.ack(worker,"execution.staged")
        if rowset(lab.db,"SELECT count(*) FROM receipt_read_ordinals")[0][0]:
            break
        advance(worker)
    else:
        raise RuntimeError("no staged pure work while a read was outstanding")
    packet=lab.relay.next()
    assert packet.path.endswith("/lookup") and not packet.upstream_done.is_set()
    before=Projection(lab.db)
    baseline=rowset(lab.db,"SELECT sequence FROM runs WHERE subject=?",(member,))[0][0]
    assert before.projection(member)["Status"]=="open" and tree not in before.pub
    lab.save("held-read")
    lab.event("held-lookup-before-pure",{"staged":event,"tree":tree,"member":member,"sequence":baseline})
    # Continue through the pure node's actual terminal commit. Its terminal
    # phase is emitted at staging, so a later publication proves real progress.
    while event["Phase"]!="terminal":
        advance(worker)
        event=lab.ack(worker,"execution.staged")
    advance(worker)
    wait_for(lambda: rowset(lab.db,"SELECT 1 FROM composition_publications WHERE operation_id=?",(tree,)),
             "pure publication with read held")
    after=Projection(lab.db)
    assert after.projection(member)["Status"]=="succeeded"
    assert rowset(lab.db,"SELECT sequence FROM runs WHERE subject=?",(member,))[0][0]>baseline
    assert not packet.upstream_done.is_set() and not rowset(lab.world,"SELECT * FROM receipt_reads")
    lab.save("pure-before-read-return")
    lab.event("pure-completed-before-release",{"member":member,"tree":tree,"outcome":after.pub[tree]})
    packet.accept.set()
    packet.respond.set()
    lab.relay.held=False
    lab.stop(worker)
    lab.service("r")
    lab.run(capacity=1)


def run_case(case, bins, out, tmp):
    lab=Lab(bins,out/case.name,tmp/case.name)
    try:
        requests=case.requests()
        args=["-left",case.assignment[0],"-right",case.assignment[1],"-left-mode",case.left,"-right-mode",case.right,
              "-right-id",str(requests[1]["id"]),"-queries","1"]
        if case.relay:
            args+=["-relay",case.relay]
        if case.deficit:
            bound={"invocations":10,"runs":10,"depth":3,"effects":1}[case.deficit]
            args+=["-"+case.deficit,str(bound)]
        if not case.recover:
            args+=["-left-observe","-right-observe"]
        lab.cli("init",*args)
        if case.recover:
            original_loss(lab)
        else:
            lab.run()
        if case.slow:
            slow_read(lab)
        else:
            lab.service("l")
            lab.service("r")
            lab.run()
        lab.save("final")
        summary=grade(case,lab)
        lab.replay()
        write(lab.out/"summary.json",summary)
        return (case.name,summary["decision"])
    finally:
        lab.close()
        trace=lab.out/"processes.json"
        trace.with_suffix(".json.gz").write_bytes(gzip.compress(trace.read_bytes(),compresslevel=5,mtime=0))
        trace.unlink()


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("bin",type=Path,help="directory containing consumer, owner and resource for this regime")
    parser.add_argument("output",type=Path)
    parser.add_argument("--case",action="append",help="run a named fixture only")
    parser.add_argument("--jobs",type=int,default=4)
    a=parser.parse_args()
    matrix=cases()
    if a.case:
        matrix=[c for c in matrix if c.name in a.case]
        if len(matrix)!=len(set(a.case)):
            parser.error("unknown case id")
    assert len({c.name for c in matrix})==len(matrix)
    dest=a.output.resolve()
    dest.mkdir(parents=True,exist_ok=False)
    bins={s:(a.bin/s).resolve() for s in ("owner","resource","consumer")}
    write(dest/"plan.json",{"cases":[asdict(c) for c in matrix],"binary_sha256":{s:sha(p) for s,p in bins.items()},"jobs":a.jobs})
    with tempfile.TemporaryDirectory(prefix="phase2d-grid-") as d:
        with ThreadPoolExecutor(max_workers=a.jobs) as pool:
            pending={pool.submit(run_case,c,bins,dest,Path(d)):c.name for c in matrix}
            results={}
            for f in as_completed(pending):
                name,status=f.result()
                results[name]=status
                print(name,status,flush=True)
    write(dest/"index.json",results)


if __name__=="__main__":
    main()
