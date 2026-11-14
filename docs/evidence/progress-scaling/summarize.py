#!/usr/bin/env python3
"""Validate the primary schedule and summarize measured resource ranges."""
import argparse
import json
from pathlib import Path
from statistics import median

from measure import ARMS, BOUNDS, compare_timing, read, sha, timing_design, write


def stats(values):
    return {'min': min(values), 'median': median(values), 'max': max(values)}


def summarize(timing):
    manifest, result = (read(timing / file) for file in ('manifest.json', 'result.json'))
    raw = timing / 'rows.jsonl'
    if result['manifest_sha256'] != sha(timing / 'manifest.json') or result['rows_sha256'] != sha(raw):
        raise ValueError('primary manifest/rows digest mismatch')
    rows = [json.loads(line) for line in raw.read_text().splitlines()]
    if manifest['lane'] != 'timing' or result['failures'] or [r['design'] for r in rows] != timing_design():
        raise ValueError('incomplete or failed primary timing design')
    for row in rows:
        design = row['design']
        expected = manifest['builds'][design['arm']]
        if (row['exit'] != 0 or row['timeout'] or row['spawn_error'] or row.get('failure') or
                row['revision'] != expected['revision'] or row['mode'] != expected['mode'] or
                row['binary_sha256'] != expected['sha256']):
            raise ValueError('unexpected primary timing outcome or build identity')
    summary = {'manifest_sha256': result['manifest_sha256'], 'rows_sha256': result['rows_sha256'],
               'warmups': len([r for r in rows if r['design']['kind'] == 'warmup']), 'bounds': {}}
    paired = compare_timing(rows)
    if paired != result['paired_timing']:
        raise ValueError('paired contrasts changed since execution')
    for bound in BOUNDS:
        arms = {}
        for arm in ARMS:
            chosen = [r for r in rows if r['design']['kind'] == 'timing' and
                      r['design']['bound'] == bound and r['design']['arm'] == arm]
            if len(chosen) != 6:
                raise ValueError('expected six measured samples per arm and bound')
            first = chosen[0]
            arms[arm] = {
                'stages': {stage: {k: stats([r['resources'][stage][k] for r in chosen]) for k in metrics}
                           for stage, metrics in first['resources'].items()},
                'operation_wall_ns': {name: stats([r['operation_wall_ns'][name] for r in chosen])
                                      for name in first['operation_wall_ns']},
                **{name: stats([r[name] for r in chosen]) for name in (
                    'execute_end_maxrss_bytes', 'process_exit_maxrss_bytes', 'driver_wall_ns_before_json',
                    'process_wrapper_seconds')},
            }
        blocks = paired[str(bound)]
        deltas = {contrast: {stage: stats([block['differences_ns'][contrast][stage] for block in blocks])
                             for stage in stages}
                  for contrast, stages in blocks[0]['differences_ns'].items()}
        summary['bounds'][str(bound)] = {'arms': arms, 'paired_differences_ns': deltas}
    return summary


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('timing', type=Path)
    parser.add_argument('output', type=Path)
    args = parser.parse_args()
    write(args.output, summarize(args.timing))
