#!/usr/bin/env python3
"""Derive nonmonotone attribution status outside the cumulative counter hook.

Input is one cumulative map[string]uint64 snapshot, not a phase delta.
"""
import argparse
import json
from pathlib import Path


def assess(snapshot):
    incomplete=snapshot.get('sql.attribution.incomplete_events',0)
    unknown_steps=sum(v for k,v in snapshot.items()
                      if k.startswith('sql.scope.unknown.') and k.endswith('.step_calls'))
    return {
        'attribution_complete': incomplete == 0,
        'definition': 'No family/cell/invalid-scope truncation; unknown is preserved separately and is not known semantic caller attribution.',
        'known_semantic_scope_attribution_complete': incomplete == 0 and unknown_steps == 0,
        'unknown_scope_step_calls': unknown_steps,
        'incomplete_events': incomplete,
    }


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--snapshot',required=True,type=Path)
    a=p.parse_args();snapshot=json.loads(a.snapshot.read_text())
    if not isinstance(snapshot,dict) or any(not isinstance(k,str) or type(v) is not int or v<0 for k,v in snapshot.items()):
        raise SystemExit('expected a cumulative map[string]uint64 snapshot')
    if 'sql.total.step_calls' not in snapshot or 'sql.attribution.incomplete_events' not in snapshot:
        raise SystemExit('missing P2 SQL snapshot fields')
    print(json.dumps(assess(snapshot),indent=2))


if __name__=='__main__':main()
