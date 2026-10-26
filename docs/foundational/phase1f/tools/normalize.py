#!/usr/bin/env python3
"""Normalize public delivered answers; never promote a rejected diagnostic candidate."""
import argparse
import hashlib
import json
from pathlib import Path


def value(record, key, missing=None):
    v=record.get(key)
    if v is None:return missing
    return v.get({'string':'Text','int':'Number','bool':'Flag'}[v['Kind']])


def citation(record,prefix=''):
    if value(record,prefix+'disposition')!='answer' or not value(record,prefix+'document'):
        return None
    return {k:value(record,prefix+k) for k in ('document','version','start','end','snippet')}


def normalize(path):
    meta=json.loads((path/'init-command.json').read_text())
    view=json.loads((path/'inspect.json').read_text())
    summary=json.loads((path/'summary.json').read_text())
    result=view['Result'] or {}
    baseline=meta['system']=='baseline'
    citations=[c for i in range(1,5) if (c:=citation(result,f'e{i}_'))] if not baseline else [c for c in [citation(result)] if c]
    initial_citations=[c for i in (1,2) if (c:=citation(result,f'e{i}_'))] if not baseline else citations
    final=dict(disposition=value(result,'disposition'),answer=value(result,'answer'),citations=citations)
    initial=final.copy() if baseline else (dict(disposition=None,answer=value(result,'initial_candidate'),citations=initial_citations)
                                         if value(result,'initial_candidate') else None)
    attempts=[json.loads(r['descriptor']) for r in view['Snapshot']['Tables']['adapter_attempts']]
    model_ids={a['Intent']['ID'] for a in attempts if a['Allowed'] and a['Prepared']['Protocol']=='openai/responses/v1'}
    results={r['Candidate']['ID']:r['Candidate'] for r in view['Results']}
    usages=[results[identity].get('Usage') for identity in model_ids if identity in results]
    known=[u for u in usages if u and all(type(u.get(k)) is int and u[k]>=0 for k in ('InputTokens','OutputTokens','TotalTokens')) and u['TotalTokens']==u['InputTokens']+u['OutputTokens']]
    usage={k:sum(u[k] for u in known) for k in ('InputTokens','OutputTokens','TotalTokens')}
    usage.update(known_results=len(known),missing_usage_results=len(usages)-len(known),
                 model_allowed_attempts=len(model_ids),unknown_model_attempts=len(model_ids-results.keys()),
                 unknown_attempts=len(view['Unknown'] or []))
    outcome=('unknown' if view['Unknown'] else 'pending') if not result else 'public-result'
    return dict(case_id=meta['case_id'],system=meta['system'],execution_outcome=outcome,
        initial=initial,final=final,diagnostic_final_candidate=value(result,'final_candidate'),
        revision_count=0 if baseline else value(result,'revision_count'),
        application_review=None if baseline else {k:value(result,k) for k in ('initial_review_decision','initial_review_reason','review_decision','review_reason','report_outcome')},
        elapsed_seconds=summary['elapsed_seconds'],usage=usage,model_requests=view['ModelRequests'],lookup_requests=view['LookupRequests'],
        estimated_token_cost_usd=summary['estimated_token_cost_usd'],unknown_cost_reservations_usd=summary['unresolved_cost_reservations'],
        unknown_ids=view['Unknown'] or [],run_identity=dict(directory=str(path),source=meta['source'],store_sha256=summary['store_sha256']),
        replay_pass=summary['replay_pass'])


def main():
    parser=argparse.ArgumentParser();parser.add_argument('directory',type=Path);args=parser.parse_args()
    configuration=json.loads((args.directory/'configuration.json').read_text())
    rows=[normalize(args.directory/(r['case_id']+'-'+r['system'])) for r in configuration['case_order']]
    (args.directory/'normalized.json').write_text(json.dumps(rows,indent=2)+'\n')
    # Deterministic opaque sorting before grading; mapping retained separately.
    ordered=sorted(rows,key=lambda r:hashlib.sha256(('phase1f-blind-v1:'+r['case_id']+':'+r['system']).encode()).hexdigest())
    packets=[];mapping=[]
    for i,row in enumerate(ordered,1):
        identity=f'response-{i:02}'
        packets.append(dict(id=identity,case_id=row['case_id'],execution_outcome=row['execution_outcome'],
                            initial=row['initial'],final=row['final'],diagnostic_final_candidate=row['diagnostic_final_candidate']))
        mapping.append(dict(id=identity,case_id=row['case_id'],system=row['system']))
    (args.directory/'blind-packets.json').write_text(json.dumps(packets,indent=2)+'\n')
    (args.directory/'blind-mapping.json').write_text(json.dumps(mapping,indent=2)+'\n')
    report=dict(runs=len(rows),model_requests=sum(r['model_requests'] for r in rows),lookup_requests=sum(r['lookup_requests'] for r in rows),
                known_input_tokens=sum(r['usage']['InputTokens'] for r in rows),known_output_tokens=sum(r['usage']['OutputTokens'] for r in rows),
                estimated_token_cost_usd=round(sum(r['estimated_token_cost_usd'] for r in rows),8),
                unknown_attempts=sum(r['usage']['unknown_attempts'] for r in rows),replay_passes=sum(r['replay_pass'] for r in rows),
                note='No quality score inferred from model review. Diagnostic candidate never substitutes for public answer; unknowns remain explicit. Latency includes controller barriers.')
    (args.directory/'totals.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report))

if __name__=='__main__':main()
