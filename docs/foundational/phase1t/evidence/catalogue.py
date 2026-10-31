"""Derive the catalogue JSONL from the archived owning-tree inspections and manifest."""

import argparse
import base64
import gzip
import hashlib
import json
from pathlib import Path
import sys


HERE = Path(__file__).resolve().parent


def read(path):
    with gzip.open(path, "rt") as f:
        return json.load(f)


def compact(value):
    return json.dumps(value, ensure_ascii=False, separators=(",", ":")).encode()


def sha(data):
    return hashlib.sha256(data).hexdigest()


def verify(condition, label):
    if not condition:
        raise ValueError(label)


class Catalogue:
    def __init__(self):
        self.rows = {}
        self.cases = []

    def add(self, kind, key, source, **values):
        rows = self.rows.setdefault(kind, {})
        row = dict(kind=kind, id=key, source=source, **values)
        if key in rows:
            verify(dict(row, source=rows[key]["source"]) == rows[key], f"inconsistent {kind} {key}")
        else:
            rows[key] = row
        return key

    def schema(self, schema, source):
        if not schema or not schema["ID"]:
            return None
        ident = sha(compact(schema["Record"]))
        verify(ident == schema["ID"], "schema ID")
        return self.add("schema", ident, source, record=schema["Record"])

    def init(self, projection, source):
        # A local audit digest for the encoded projection, not a runtime-assigned ID.
        return self.add("initializer", sha(compact(projection)), source, projection=projection)

    def program(self, ident, raw, source):
        verify(sha(raw) == ident, "program digest")
        program = json.loads(raw)
        metadata = program.get("Author") or {}
        invocations = metadata.get("Invocations") or {}
        calls = []
        for pc, ins in enumerate(program["Code"]):
            effect = ins["Effect"]
            if not effect or effect["Kind"] != "invocation":
                continue
            role = effect["Obligation"]["Literal"]["Text"]
            decl = invocations[role]
            verify(effect["Target"]["Literal"]["Text"] == decl["TemplateID"], "original declaration")
            calls.append(
                dict(
                    pc=pc,
                    role=role,
                    operation_register=ins["Dst"],
                    template=decl["TemplateID"],
                    request_schema=self.schema(decl["Input"], source),
                    report_schema=self.schema(decl["Output"], source),
                )
            )
        verify({c["role"] for c in calls} == invocations.keys(), "declaration coverage")
        ports = {
            name: dict(receive=self.schema(port["Receive"], source), send=self.schema(port["Send"], source))
            for name, port in metadata.get("Ports", {}).items()
        }
        receives = {pc: self.schema(schema, source) for pc, schema in (metadata.get("Receives") or {}).items()}
        self.add(
            "program",
            ident,
            source,
            name=metadata.get("Name"),
            semantics=program["Semantics"],
            compiler=metadata.get("Compiler"),
            insns=len(program["Code"]),
            inputs=self.schema(metadata.get("Inputs"), source),
            outputs=self.schema(metadata.get("Outputs"), source),
            ports=ports,
            receives=receives,
            calls=calls,
        )

    def template(self, tmpl, source):
        tid = tmpl["ID"]
        verify(sha(compact([tmpl["Version"], dict(tmpl, ID="")])) == tid, "retained template ID")
        member = tmpl["EntryMember"]
        verify(set(tmpl["Members"]) == {member}, "non-singleton template")
        entry = tmpl["Members"][member]
        pid = entry["ProgramID"]
        program = self.rows["program"][pid]
        port = program["ports"][tmpl["EntryPort"]]
        children = [call["template"] for call in program["calls"]]
        verify(sorted(set(children)) == (tmpl["Templates"] or []), "template scope catalogue")
        self.add(
            "template",
            tid,
            source,
            member=member,
            entry_port=tmpl["EntryPort"],
            caller_peer=tmpl["CallerPeer"],
            program=pid,
            initializer=self.init(entry["Initial"], source),
            request_schema=port["receive"],
            report_schema=port["send"],
            children=children,
        )

    def case(self, inspection, source, label, manifest=None):
        t = inspection["Tables"]
        raw_programs = {r["id"]: r["bytes"].encode() for r in t["programs"]}
        templates = {r["id"]: json.loads(r["descriptor"]) for r in t["composition_templates"]}
        (root,) = inspection["Roots"]
        deploy = json.loads(t["composition_deployment"][0]["descriptor"])
        if manifest:
            tree, prov = manifest["Tree"], manifest["Provision"]
            verify(
                prov["Runs"] == deploy["Runs"] and prov["Subjects"] == deploy["Subjects"], "consumer fixed runs"
            )
            verify(
                raw_programs
                == {p["ID"]: base64.b64decode(p["Bytes"], validate=True) for p in prov["Programs"]},
                "consumer programs",
            )
            verify(
                templates
                == {v["ID"]: v for v in prov["Composition"]["Templates"]}
                == {v["ID"]: v for v in [tree["Descriptor"], *tree["Dependencies"]]},
                "consumer templates",
            )
        for pid, raw in raw_programs.items():
            self.program(pid, raw, source)
        for template in templates.values():
            self.template(template, source)
        installed = set()

        def visit(tid):
            if tid not in installed:
                installed.add(tid)
                for child in self.rows["template"][tid]["children"]:
                    visit(child)

        fixed = []
        for r in deploy["Runs"]:
            pid = r["ProgramID"]
            verify(pid in raw_programs, "fixed program missing")
            fixed.append(dict(run=r["ID"], program=pid, initializer=self.init(r["Projection"], source)))
            for call in self.rows["program"][pid]["calls"]:
                visit(call["template"])
        verify(installed == templates.keys(), "installed vs reachable closure")
        verify(
            raw_programs.keys()
            == {r["program"] for r in fixed} | {self.rows["template"][k]["program"] for k in installed},
            "program closure",
        )
        requests = {json.loads(r["descriptor"])["ID"]: r["template_id"] for r in t["composition_invocations"]}
        for r in t["composition_members"]:
            d = json.loads(r["descriptor"])
            template = self.rows["template"][requests[d["InvocationID"]]]
            verify(
                template["program"] == d["ProgramID"]
                and template["initializer"] == self.init(d["Initial"], source),
                "admitted member identity",
            )
        (top,) = self.rows["program"][next(r["program"] for r in fixed if r["run"]["Subject"] == "root")][
            "calls"
        ]
        case = dict(
            kind="case",
            label=label,
            source=source,
            namespace=inspection["Namespace"],
            root_id=root["RootID"],
            limits=root["Limits"],
            fixed=fixed,
            tree=top["template"],
        )
        if manifest:
            verify(
                top["template"] == tree["Descriptor"]["ID"]
                and top["request_schema"] == tree["Input"]["ID"]
                and top["report_schema"] == tree["Output"]["ID"],
                "consumer root template and schemas",
            )
            case["manifest"] = "consumer-manifest.json.gz"
        self.cases.append(case)

    def lines(self):
        for kind in ("schema", "initializer", "program", "template"):
            for key, row in sorted(self.rows[kind].items()):
                yield compact(row).decode()
        for row in self.cases:
            yield compact(row).decode()


def derive():
    catalogue = Catalogue()
    for group, basename in (("automatic", "case.json.gz"), ("manual", "manual.json.gz")):
        paths = sorted((HERE / "runs" / group).glob("*/" + basename))
        verify(len(paths) == {"automatic": 80, "manual": 14}[group], "incomplete archived domain")
        for path in paths:
            raw = read(path)
            label = raw["shape"] + "/" + raw["assignment"] if group == "automatic" else raw["kind"]
            catalogue.case(raw["inspection"], path.relative_to(HERE).as_posix(), label)
    name = "consumer-execution.json.gz"
    run = json.loads(read(HERE / name)["output"])
    manifest = json.loads(read(HERE / "consumer-manifest.json.gz")["output"])
    catalogue.case(run["Inspection"], name, run["Term"], manifest)
    return catalogue


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="verify existing catalogue.jsonl")
    args = parser.parse_args()
    catalogue = derive()
    data = ("\n".join(catalogue.lines()) + "\n").encode()
    if args.check:
        verify((HERE / "catalogue.jsonl").read_bytes() == data, "catalogue differs")
        counts = {k: len(v) for k, v in catalogue.rows.items()}
        print(json.dumps(dict(cases=len(catalogue.cases), **counts, sha256=sha(data)), sort_keys=True))
    else:
        sys.stdout.buffer.write(data)


if __name__ == "__main__":
    main()
