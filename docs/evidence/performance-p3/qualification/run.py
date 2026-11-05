#!/usr/bin/env python3
"""Frozen-route P3 qualification (Python 3.12+). Each phase runs once in a fresh root."""
import argparse
import difflib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile

from common import Recorder, copy_db, inventory, members, sha, write

BASE = '926a899b651fccc48be52a054612a92afcf1ddfd'
CANDIDATE = '49df59f1f767dd63fec5e238bf565d023261a75b'
OLD = 'af6943a5805a9a6c0e8fe77f804264f4765d07ec'
SERIES = '^TestWeatherSeriesProcess$'
CONTINUATION = '^TestWeatherContinuationProcess$/(before_approval_commit|after_approval_commit|after_successor_commit|accepted_lost_response|retained_before_settlement|settled_before_consumption|after_terminal)$'
RECOVERY = '^TestWeatherProcessRecovery$/(after_request_commit|two_interrupted_recoveries|accepted_and_lost_then_explicitly_reobserved|completed_then_explicitly_refreshed)$'
PUBLIC = '^TestWeatherSeries(Eight|OriginalAndLegacyPrograms|Corruption)$'
AUTHORITY = '^TestRequestAuthority'
DEBT = '^TestCompetingReadCannotSettleAdopterDebt$'
CONSUMER = '^Test(FreshStorageFormats|HistoricalArchivesRemainReadOnly|NewSeriesFixture|CanonicalLoweringEquality)$'


def freeze(root, repo):
    root.mkdir(parents=True, exist_ok=False)
    env = {key: os.environ[key] for key in ('PATH', 'HOME', 'TMPDIR', 'SDKROOT', 'SHELL') if key in os.environ}
    env.update(CGO_ENABLED='1', GOCACHE='/private/tmp/alan-p3-j216kxoz/go-cache',
               GOGC='100', GOMAXPROCS='4', GOMEMLIMIT='off', GOPROXY='off', GOSUMDB='off',
               GOTOOLCHAIN='local', GOWORK='off', GOFLAGS='', GODEBUG='', GORACE='',
               GOTRACEBACK='single', LANG='C.UTF-8', LC_ALL='C.UTF-8', PYTHONDONTWRITEBYTECODE='1')
    tools = root / 'controller'
    shutil.copytree(Path(__file__).parent, tools, ignore=shutil.ignore_patterns('__pycache__'))
    routes = {'weather_series': SERIES, 'weather_continuation': CONTINUATION,
              'weather_recovery': RECOVERY, 'weather_public': PUBLIC,
              'authority_ordinary_and_race': AUTHORITY, 'debt_ordinary_and_race': DEBT,
              'external_ordinary_and_race': CONSUMER,
              'direct': ['success/reopen0', 'success/reopen1', 'refusal3/reopen0', 'refusal3/reopen1'],
              'http': ['success8/reopen0', 'success8/reopen1', 'original verify_tools', 'original prove_comparison',
                       'matching contact identity/capability/effect/protocol/endpoint mutations'],
              'archive_readers': '31 fresh series archives per source (10 raw stopped before readers +21 existing stopped/final/reference), exact public+History+Replay+typed SQL on complete copies',
              'opposite_build': 'both writer directions, staged/pre-commit and committed/post-commit; ACK+SIGKILL+reap',
              'historical': 'fresh9/11, historical4/9, genuine11->12, baseline generic11/12, old rejection13/14',
              'bounds': 'no timing, research, Phase3B4, authority×series, or production changes'}
    plan = {'version': 'alan/p3-task4-routes/v1', 'repo': str(repo), 'environment': env,
            'sources': {'baseline': BASE, 'candidate': CANDIDATE, 'old': OLD}, 'routes': routes,
            'controller': inventory(tools), 'same_seed_normalization': ['source_label', 'store', 'physical SQL row order'],
            'weather_archive_overlay': 'copy-only helpers preserve every main/WAL/SHM before stopped readers, before/copy/after hashes and public consumer executable/module; assertions unchanged'}
    write(root / 'route-manifest.json', plan)
    for label, commit in plan['sources'].items():
        dest = root / label
        dest.mkdir()
        archive = root / (label + '.tar')
        with archive.open('xb') as out:
            subprocess.run(['git', '-C', str(repo), 'archive', commit], stdout=out, check=True, env=env)
        with tarfile.open(archive) as tar:
            tar.extractall(dest, filter='data')
        tree = subprocess.check_output(['git', '-C', str(repo), 'rev-parse', commit + '^{tree}'], env=env).decode().strip()
        write(root / (label + '-source.json'), {'commit': commit, 'tree': tree, 'archive_sha256': sha(archive), 'files': inventory(dest)})
    # Hardened reader source must remain byte-identical to actual production baseline.
    rel = 'internal/durabletest/competing_debt_test.go'
    if sha(root / 'baseline' / rel) != sha(root / 'candidate' / rel):
        raise ValueError('hardened competing-reader test changed')
    write(root / 'freeze.json', {'route_manifest_sha256': sha(root / 'route-manifest.json'),
                                 'hardened_reader_sha256': sha(root / 'candidate' / rel)})


def build(r):
    root = r.root
    r.run('go-version', ['go', 'version'])
    r.run('go-environment', ['go', 'env', '-json', 'GOVERSION', 'GOOS', 'GOARCH', 'CGO_ENABLED', 'CC', 'GOMODCACHE', 'GOCACHE', 'GOTOOLCHAIN', 'GOPROXY', 'GOSUMDB'])
    for label in ('baseline', 'candidate', 'old'):
        src = root / label
        r.run('build-' + label + '-cli', ['go', 'build', '-buildvcs=false', '-o', root / (label + '-alan'), './cmd/alan'], src)
    for label in ('baseline', 'candidate'):
        src = root / label
        for kind, directory in [('direct', 'series-scaling'), ('http', 'performance-p2')]:
            r.run('build-' + label + '-' + kind, ['sh', root / 'candidate' / 'docs/evidence' / directory / 'differential/build.sh', src, root / (label + '-' + kind)])
        # Explicitly retained ordinary/race binaries allow build identity inspection.
        for package, folder in [('weather', 'examples/weather-agent'), ('authority', 'internal/executor'), ('debt', 'internal/durabletest')]:
            profiles = ('ordinary',) if package == 'weather' else ('ordinary', 'race')
            for profile in profiles:
                argv = ['go', 'test', '-c', '-buildvcs=false', '-tags=integration']
                if profile == 'race':
                    argv += ['-race']
                if package == 'weather':
                    argv += ['-overlay=' + str(weather_overlay(r, label))]
                argv += ['-o', root / (label + '-' + package + '-' + profile + '.test'), './' + folder]
                r.run('build-' + label + '-' + package + '-' + profile, argv, src)
    write(root / 'binary-identities.json', {p.name: sha(p) for p in root.iterdir() if p.is_file() and (p.name.endswith('.test') or p.name.endswith(('-alan', '-direct', '-http')))})


def weather_overlay(r, label):
    root = r.root
    out = root / ('weather-overlay-' + label)
    out.mkdir()
    source = root / label / 'examples/weather-agent/series_process_test.go'
    text = source.read_text()
    start, end = text.index('func preserveSeries('), text.index('func recordSeries(')
    helper = '''func preserveSeries(t *testing.T, db, name string) {
 t.Helper()
 dir := os.Getenv("ALAN_SERIES_EVIDENCE_DIR")
 if dir == "" { return }
 if err := os.MkdirAll(dir, 0700); err != nil { t.Fatal(err) }
 capture := func(path string) map[string]string {
  hashes := map[string]string{}
  for _, suffix := range []string{"", "-wal", "-shm"} {
   b, err := os.ReadFile(path + suffix)
   if os.IsNotExist(err) && suffix != "" { continue }
   if err != nil { t.Fatal(err) }
   hashes[suffix] = fmt.Sprintf("%x", sha256.Sum256(b))
  }
  return hashes
 }
 before := capture(db)
 target := filepath.Join(dir, name+".sqlite")
 for suffix := range before {
  b, err := os.ReadFile(db + suffix); if err != nil { t.Fatal(err) }
  f, err := os.OpenFile(target+suffix, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600); if err != nil { t.Fatal(err) }
  if _, err = f.Write(b); err != nil { t.Fatal(err) }; if err = f.Close(); err != nil { t.Fatal(err) }
 }
 copied, after := capture(target), capture(db)
 equal := reflect.DeepEqual(before, copied) && reflect.DeepEqual(before, after)
 proof, err := json.Marshal(map[string]any{"source":db,"target":target,"source_before":before,"copy":copied,"source_after":after,"equal":equal})
 if err != nil { t.Fatal(err) }
 if err = os.WriteFile(target+".copy.json", proof, 0600); err != nil { t.Fatal(err) }
 if !equal { t.Fatal("archive source/copy changed") }
}
'''
    patched = text[:start] + helper + text[end:]
    patched = patched.replace('owner.kill(t, witness)\n', 'owner.kill(t, witness)\n\t\t\tpreserveSeries(t, db, tc.name+"-raw-stop")\n', 1)
    patched = patched.replace('import (', 'import (\n "crypto/sha256"', 1)
    replacement = out / source.name
    replacement.write_text(patched)
    subprocess.run(['gofmt', '-w', str(replacement)], check=True, env=r.env)
    (out / 'series.patch').write_text(''.join(difflib.unified_diff(text.splitlines(True), replacement.read_text().splitlines(True), fromfile=str(source), tofile=str(replacement))))
    public_source = source.with_name('process_test.go')
    public_text = public_source.read_text()
    retention = '''	if archive := os.Getenv("ALAN_P3_CONSUMER_EVIDENCE_DIR"); archive != "" {
        dest := filepath.Join(archive, t.Name())
        if err := os.MkdirAll(dest, 0700); err != nil { t.Fatal(err) }
        entries, err := os.ReadDir(dir); if err != nil { t.Fatal(err) }
        for _, entry := range entries {
            if entry.IsDir() { continue }
            b, err := os.ReadFile(filepath.Join(dir, entry.Name())); if err != nil { t.Fatal(err) }
            f, err := os.OpenFile(filepath.Join(dest, entry.Name()), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700); if err != nil { t.Fatal(err) }
            if _, err = f.Write(b); err != nil { t.Fatal(err) }; if err = f.Close(); err != nil { t.Fatal(err) }
        }
        t.Logf("P3_PUBLIC_CONSUMER %s", dest)
    }
'''
    if public_text.count('\treturn binary\n') != 1:
        raise ValueError('public consumer retention anchor changed')
    public_replacement = out / public_source.name
    public_replacement.write_text(public_text.replace('\treturn binary\n', retention + '\treturn binary\n'))
    subprocess.run(['gofmt', '-w', str(public_replacement)], check=True, env=r.env)
    (out / 'public-consumer.patch').write_text(''.join(difflib.unified_diff(public_text.splitlines(True), public_replacement.read_text().splitlines(True), fromfile=str(public_source), tofile=str(public_replacement))))
    write(out / 'overlay.json', {'Replace': {str(source): str(replacement), str(public_source): str(public_replacement)}})
    write(out / 'identity.json', {'original_sha256': sha(source), 'replacement_sha256': sha(replacement),
                                'public_original_sha256': sha(public_source), 'public_replacement_sha256': sha(public_replacement), 'scope': 'archive helpers and pre-reader copy only; no crash/semantic assertions changed'})
    return out / 'overlay.json'


def weather(r):
    for label in ('baseline', 'candidate'):
        binary = r.root / (label + '-weather-ordinary.test')
        src = r.root / label / 'examples/weather-agent'
        archives = r.root / (label + '-archives')
        for name, pattern in [('series', SERIES), ('continuation', CONTINUATION), ('recovery', RECOVERY), ('public', PUBLIC)]:
            r.run(label + '-weather-' + name, [binary, '-test.v', '-test.count=1', '-test.timeout=15m', '-test.run=' + pattern], src,
                  dict({'ALAN_SERIES_EVIDENCE_DIR': str(archives)} if name == 'series' else {}, ALAN_P3_CONSUMER_EVIDENCE_DIR=str(r.root / (label + '-public-binaries'))))
        write(r.root / (label + '-archives-seal.json'), inventory(archives))


def compare(r, name, reports, dbs, kind='direct'):
    directory = 'series-scaling' if kind == 'direct' else 'performance-p2'
    tools = r.root / 'candidate/docs/evidence' / directory / 'differential'
    path = r.run(name, ['python3', tools / 'compare.py', 'same-seed', *reports, '--baseline-db', dbs[0], '--candidate-db', dbs[1]])
    if not json.loads(path.read_text()).get('equal_for_stated_coverage'):
        raise ValueError('comparator did not affirm equality')


def archives(r):
    root = r.root
    out = root / 'archive-readers'
    out.mkdir()
    for label in ('baseline', 'candidate'):
        archive = root / (label + '-archives')
        for source in sorted(archive.glob('*.sqlite')):
            name = 'archive-' + label + '-' + source.stem
            dbs, reports = [], []
            for reader in ('baseline', 'candidate'):
                db = out / name / reader / 'store.sqlite'
                copy_db(source, db, out / (name + '-' + reader + '-copy.json'))
                dbs.append(db)
                reports.append(r.run(name + '-' + reader, [root / (reader + '-direct'), '-mode', 'generic-reader', '-db', db, '-source-label', r.plan['sources'][reader]]))
            compare(r, name + '-compare', reports, dbs)
        if inventory(archive) != json.loads((root / (label + '-archives-seal.json')).read_text()):
            raise ValueError('sealed original changed')


def authority(r):
    root = r.root
    for label in ('baseline', 'candidate'):
        for profile in ('ordinary', 'race'):
            for package, pattern in [('authority', AUTHORITY), ('debt', DEBT)]:
                r.run(label + '-' + package + '-' + profile,
                      [root / (label + '-' + package + '-' + profile + '.test'), '-test.v', '-test.count=1', '-test.timeout=10m', '-test.run=' + pattern],
                      root / label / ('internal/executor' if package == 'authority' else 'internal/durabletest'),
                      {'ALAN_PHASE3B3_OLD_READER': str(root / 'old-alan')})
        r.run(label + '-runtime-authority', ['go', 'test', '-json', '-count=1', '-tags=integration', '-run=^TestRequestAuthorityPublicCommittedControl$', './runtime'], root / label)


def external(r):
    root = r.root
    out = root / 'external'
    out.mkdir()
    candidate, baseline = root / 'candidate', root / 'baseline'
    source = candidate / 'docs/evidence/bounded-series/performance/module'
    module, oldmodule = out / 'candidate-module', out / 'baseline-reader-module'
    shutil.copytree(source, module)
    shutil.copytree(source, oldmodule)
    r.run('external-bind-candidate', ['go', 'mod', 'edit', '-replace=github.com/JakeFAU/alan=' + str(candidate)], module)
    for profile in ('ordinary', 'race'):
        fixtures = out / (profile + '-fixtures')
        fixtures.mkdir()
        binary = out / (profile + '-consumer.test')
        command = ['go', 'test', '-mod=readonly', '-c', '-tags=series', '-buildvcs=false', '-o', binary]
        if profile == 'race':
            command.insert(2, '-race')
        r.run('external-' + profile + '-build', command, module)
        r.run('external-' + profile, [binary, '-test.run=' + CONSUMER, '-test.v', '-test.count=1', '-test.timeout=10m'], module,
              {'ALAN_SOURCE': str(candidate), 'SERIES_FIXTURE_DIR': str(fixtures)})
    ordinary = out / 'ordinary-fixtures'
    copy_db(ordinary / 'series11.sqlite', ordinary / 'series12.sqlite', out / 'promotion-input-copy.json')
    overlay = candidate / 'docs/evidence/bounded-series/performance/overlay/promote_series_test.go.txt'
    write(out / 'promotion-overlay.json', {'Replace': {str(candidate / 'internal/executor/independent_series_promotion_test.go'): str(overlay)}})
    r.run('external-genuine-promotion', ['go', 'test', '-overlay=' + str(out / 'promotion-overlay.json'), '-run=^TestIndependentPromoteSeries$', '-v', '-count=1', './internal/executor'], candidate, {'SERIES_PROMOTE_PATH': str(ordinary / 'series12.sqlite')})
    r.run('external-bind-reader', ['go', 'mod', 'edit', '-replace=github.com/JakeFAU/alan=' + str(baseline)], oldmodule)
    reader = out / 'baseline-reader'
    r.run('external-build-reader', ['go', 'build', '-mod=readonly', '-buildvcs=false', '-o', reader, './cmd/base-reader'], oldmodule)
    for version in (11, 12):
        db = out / ('reader' + str(version)) / 'store.sqlite'
        copy_db(ordinary / ('series' + str(version) + '.sqlite'), db, out / ('reader' + str(version) + '-copy.json'))
        r.run('external-baseline-read-' + str(version), [reader, db], oldmodule)
    # TestRetainedSeriesFormats performs read-only inspection/replay; run on complete
    # isolated copies too, preserving promotion artifacts as original evidence.
    retained = out / 'retained-copies'
    for version in (11, 12):
        copy_db(ordinary / ('series' + str(version) + '.sqlite'), retained / ('series' + str(version) + '.sqlite'), out / ('retained' + str(version) + '-copy.json'))
    r.run('external-retained-formats', [out / 'ordinary-consumer.test', '-test.run=^TestRetainedSeriesFormats$', '-test.v', '-test.count=1'], module,
          {'ALAN_SOURCE': str(candidate), 'SERIES_FIXTURE_DIR': str(retained)})
    write(out / 'binaries.json', {p.name: sha(p) for p in (out / 'ordinary-consumer.test', out / 'race-consumer.test', reader)})


def seal(r):
    for label in ('baseline', 'candidate', 'old'):
        actual = inventory(r.root / label)
        write(r.root / (label + '-source-after.json'), actual)
        if actual != json.loads((r.root / (label + '-source.json')).read_text())['files']:
            raise ValueError('immutable source changed: ' + label)
    write(r.root / 'raw-index.json', inventory(r.root))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('phase', choices=('freeze', 'build', 'weather', 'archives', 'authority', 'external', 'seal'))
    p.add_argument('--root', type=Path, required=True)
    p.add_argument('--repo', type=Path, default=Path.cwd())
    a = p.parse_args()
    if a.phase == 'freeze':
        freeze(a.root.resolve(), a.repo.resolve())
    else:
        r = Recorder(a.root)
        globals()[a.phase](r)


if __name__ == '__main__':
    main()
