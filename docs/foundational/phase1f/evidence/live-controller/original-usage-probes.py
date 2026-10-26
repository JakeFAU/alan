"""Original review accounting probes; no descriptor/network/credential operations."""
import sys,json,copy
from pathlib import Path
sys.path.insert(0,'docs/foundational/phase1f/tools')
import live_control as c
v=json.loads(Path('docs/foundational/phase1f/evidence/live-phase1e/inspect.json').read_text())
for name,usage in [('negative-total',{'InputTokens':0,'OutputTokens':0,'TotalTokens':-1,'Details':{}}),('inconsistent-total',{'InputTokens':1,'OutputTokens':1,'TotalTokens':666,'Details':{}})]:
    view=copy.deepcopy(v)
    candidate={'ID':'probe','ReturnedModel':c.MODEL,'Response':'{"service_tier":"default"}','Usage':usage,'RequestID':'probe','ResponseID':'probe'}
    view['Snapshot']['Tables']['adapter_results']=[{'descriptor':json.dumps({'Candidate':candidate})}]
    run={'operations':[{'id':'probe','reserved_usd':.2}]}
    c.reconcile(run,view)
    print(name,json.dumps(run), 'charge=',c.charge({'controlled_runs':[run]}))
