#!/usr/bin/env python3
"""Exercise the README examples with fresh stores and an ephemeral endpoint port."""
from pathlib import Path
import gzip, hashlib, json, os, subprocess, tempfile
ROOT = Path(__file__).resolve().parents[4]
DOC = ROOT / 'docs/foundational/phase1d1'
source = json.loads((DOC / 'source-manifest.json').read_text())['source_sha256']
env = dict(os.environ, GOTOOLCHAIN='local', GOCACHE='/tmp/alan-phase1d1-go-build', GOPROXY='off', GOSUMDB='off')
records = []
def run(args):
    p = subprocess.run(args, cwd=ROOT, env=env, text=True, capture_output=True, timeout=60)
    records.append(dict(Command=[str(x) for x in args], ExitCode=p.returncode, Stdout=p.stdout, Stderr=p.stderr))
    assert p.returncode == 0, records[-1]
    return p.stdout
with tempfile.TemporaryDirectory(prefix='alan-readme-') as name:
    tmp = Path(name); alan = tmp / 'alan'; endpoint = tmp / 'alan-resource'
    run(['go','build','-buildvcs=false','-o',alan,'./cmd/alan'])
    run(['go','build','-buildvcs=false','-o',endpoint,'./cmd/alan-resource'])
    for demo, extra in [('nested', []), ('exhausted', ['--retry-budget','1','--successor-input','fail'])]:
        db = tmp / (demo + '.sqlite')
        run([alan,'init','--db',db,'--demo','nested',*extra])
        report = json.loads(run([alan,'run','--db',db]))
        assert report['Quiescent'] and not report['OpenRuns'], report
        run([alan,'inspect','--db',db]); run([alan,'replay','--db',db])
        run([alan,'run','--db',db]); run([alan,'replay','--db',db])
    p = subprocess.Popen([endpoint,'--db',tmp/'resource.sqlite','--token','readme-smoke','--listen','127.0.0.1:0'], cwd=ROOT, env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    try:
        ready = json.loads(p.stdout.readline()); records.append(dict(EndpointReady=ready))
        db = tmp/'external.sqlite'
        run([alan,'init','--db',db,'--demo','external','--endpoint',ready['URL'],'--profile','idempotent'])
        report = json.loads(run([alan,'run','--db',db,'--max-in-flight','2','--external-timeout','2s']))
        assert report['Quiescent'] and not report['OpenRuns'] and not report['UnknownExternal'], report
        run([alan,'inspect','--db',db]); run([alan,'replay','--db',db])
    finally:
        p.terminate(); stdout,stderr=p.communicate(timeout=10)
        records.append(dict(EndpointExit=p.returncode, Stdout=stdout, Stderr=stderr, Expected='controller SIGTERM after smoke completion'))
raw = ''.join(json.dumps(dict(SourceID=source, **r))+'\n' for r in records).encode()
out=DOC/'evidence/readme-smoke.jsonl.gz'; out.write_bytes(gzip.compress(raw,mtime=0))
print(json.dumps(dict(SourceID=source, Commands=sum('Command' in r for r in records), Result='PASS', SHA256=hashlib.sha256(raw).hexdigest())))
