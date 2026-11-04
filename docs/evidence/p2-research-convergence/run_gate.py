#!/usr/bin/env python3
"""Run one predeclared gate; keep original outputs and tracked-source custody."""
import argparse
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import time


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def inventory(root):
    paths = subprocess.check_output(['git', '-C', str(root), 'ls-files', '-z']).split(b'\0')
    return {os.fsdecode(p): digest(root / os.fsdecode(p)) for p in paths if p}


def write(path, value):
    path.write_text(json.dumps(value, sort_keys=True, indent=2) + '\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('plan')
    parser.add_argument('name')
    args = parser.parse_args()
    plan_path = pathlib.Path(args.plan).resolve()
    plan = json.loads(plan_path.read_text())
    command = next(x for x in plan['commands'] if x['name'] == args.name)
    out = pathlib.Path(plan['raw_root']) / 'gates' / args.name
    out.mkdir(parents=True, exist_ok=False)
    roots = [pathlib.Path(x) for x in plan['source_roots']]
    before = {str(root): inventory(root) for root in roots}
    write(out / 'source-before.json', before)
    env = os.environ.copy()
    env.update(plan['environment'])
    env.update(command.get('environment', {}))
    tool = pathlib.Path(shutil.which(command['argv'][0], path=env.get('PATH'))).resolve()
    visible_keys = set(plan['environment']) | set(command.get('environment', {})) | {
        'PATH', 'TMPDIR', 'TMP', 'TEMP', 'HOME', 'SHELL', 'LANG', 'LC_ALL',
        'LC_CTYPE', 'TZ', 'SDKROOT', 'DEVELOPER_DIR', 'CC', 'CXX',
    }
    recorded_env = {key: env[key] for key in sorted(visible_keys) if key in env}
    omitted_keys = sorted(set(env) - visible_keys)
    record = dict(command, plan=str(plan_path), plan_sha256=digest(plan_path),
                  runner_sha256=digest(pathlib.Path(__file__)),
                  environment=recorded_env, omitted_environment_variable_names=omitted_keys,
                  executable=str(tool), executable_sha256=digest(tool),
                  started_utc=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()))
    write(out / 'command.json', record)
    start = time.monotonic()
    timed_out = False
    with (out / 'stdout.log').open('wb') as stdout, (out / 'stderr.log').open('wb') as stderr:
        try:
            result = subprocess.run(command['argv'], cwd=command['cwd'], env=env,
                                    stdout=stdout, stderr=stderr, timeout=command.get('timeout_seconds', 2400))
            code = result.returncode
        except subprocess.TimeoutExpired:
            timed_out, code = True, None
    after = {str(root): inventory(root) for root in roots}
    write(out / 'source-after.json', after)
    events = []
    for line in (out / 'stdout.log').read_text(errors='replace').splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if isinstance(event, dict) and event.get('Action') in ('pass', 'fail', 'skip'):
            events.append(event)
    write(out / 'terminal-events.json', events)
    record.update(exit=code, timed_out=timed_out, elapsed_seconds=time.monotonic()-start,
                  sources_unchanged=before == after,
                  expected_exit=command.get('expected_exit', 0),
                  stdout_sha256=digest(out/'stdout.log'), stderr_sha256=digest(out/'stderr.log'))
    record['ok'] = code == record['expected_exit'] and not timed_out and before == after
    write(out / 'result.json', record)
    print(json.dumps({key: record[key] for key in ('name', 'exit', 'timed_out', 'ok', 'elapsed_seconds')}), flush=True)
    return 0 if record['ok'] else 1


if __name__ == '__main__':
    raise SystemExit(main())
