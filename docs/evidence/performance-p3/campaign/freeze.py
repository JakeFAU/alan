#!/usr/bin/env python3
"""Export and build the exact two inputs, then bind the untouched P3 design. No runs."""
import io
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
from custody import P3, REPO, environment, inventory, sha, tools, write, validate_plan, command, bound_row


def main():
    root = Path(sys.argv[1]).resolve()
    root.mkdir(parents=True, exist_ok=False)
    env = environment()
    commands = []
    def run(argv, cwd):
        record = {'argv': list(map(str, argv)), 'cwd': str(cwd), 'environment': env}
        result = subprocess.run(record['argv'], cwd=cwd, env=env, capture_output=True)
        record.update(exit=result.returncode, stdout=result.stdout.decode(), stderr=result.stderr.decode())
        commands.append(record)
        # Persist failures too, before examining output or raising.
        (root / 'build-commands.json').write_text(json.dumps(commands, indent=2) + '\n')
        result.check_returncode()
        return result.stdout
    go_binary = '/opt/homebrew/bin/go'
    go_before = sha(go_binary)
    module_cache = run([go_binary, 'env', 'GOMODCACHE'], root).decode().strip()
    dependency_dir = str(Path(module_cache) / 'github.com/mattn/go-sqlite3@v1.14.52')
    dependency_before = inventory(dependency_dir)
    variants = {}
    for variant, commit in [('A', '926a899b651fccc48be52a054612a92afcf1ddfd'), ('B', '4f5fe2cc83316900a2f387180dd9f856db5303a0')]:
        source = root / (variant + '-source')
        source.mkdir()
        archive = subprocess.check_output(['git', '-C', str(REPO), 'archive', commit])
        (root / (variant + '-source.tar')).write_bytes(archive)
        with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
            for member in tar.getmembers():
                if not (member.isfile() or member.isdir()) or Path(member.name).is_absolute() or '..' in Path(member.name).parts:
                    raise ValueError('unsafe Git archive member: ' + member.name)
            tar.extractall(source)
        module = root / (variant + '-driver')
        shutil.copytree(P3.parent / 'performance-p2' / 'driver', module)
        # The source driver files are byte-identical. Retain the exact modfile
        # replacement rather than hiding it in build.sh's deleted temporary dir.
        run(['/opt/homebrew/bin/go', 'mod', 'edit', '-replace=github.com/JakeFAU/alan=' + str(source)], module)
        module_before = inventory(module)
        binary = root / (variant + '.bin')
        before = inventory(source)
        run(['/opt/homebrew/bin/go', 'build', '-mod=readonly', '-buildvcs=false', '-o', str(binary), '.'], module)
        if before != inventory(source) or module_before != inventory(module) or go_before != sha(go_binary) or dependency_before != inventory(dependency_dir):
            raise ValueError('build changed source')
        build_info = run(['/opt/homebrew/bin/go', 'version', '-m', str(binary)], root).decode()
        variants[variant] = {'commit': commit, 'tree': subprocess.check_output(['git', '-C', str(REPO), 'rev-parse', commit + '^{tree}'], text=True).strip(),
                             'source_dir': str(source), 'source_files': before, 'archive_sha256': sha(root / (variant + '-source.tar')),
                             'module_dir': str(module), 'module_files': inventory(module), 'source_before_build':before, 'module_before_build':module_before, 'binary': str(binary),
                             'binary_sha256': sha(binary), 'go_build_info': build_info}
    version = json.loads(run(['/opt/homebrew/bin/go', 'env', '-json', 'GOVERSION', 'GOOS', 'GOARCH'], root))
    dependencies = run(['/opt/homebrew/bin/go', 'list', '-m', '-json', 'all'], Path(variants['B']['module_dir'])).decode()
    plan = {'version':'alan/performance-p3/binding-v1', 'design':json.loads((P3/'schedule.json').read_text()),
            'variants': variants, 'environment':env, 'cwd':str(root), 'output':str(root/'campaign'),
            'python':str(Path(sys.executable).resolve()), 'python_sha256':sha(sys.executable),
            'tool_hashes':tools(), 'go_binary':go_binary, 'go_binary_sha256':go_before, 'dependencies':dependencies,
            'dependency_dir':dependency_dir, 'dependency_files':dependency_before,
            'runtime':{'go_version':version['GOVERSION'], 'goos':version['GOOS'], 'goarch':version['GOARCH'],
                       'gomaxprocs':4, 'mem_profile_rate':524288, 'profiled':False, 'instrumented':False,
                       'schedule':'ordinary', 'reopen_every':0, 'prefix_enabled':True},
            'controls':{'source_files':variants['B']['source_files'], 'selector_probe_sha256':sha(P3/'selector_probe.py'),
                        'cases':['32 runs / 4 programs / all waiting', '32 runs / 32 programs / all waiting', '32 runs / 4 programs / last ready'],
                        'classification':'separate diagnostic; never primary timing', 'status':'source frozen; not executed'}}
    plan['commands'] = [command(plan,bound_row(plan,r)) for r in plan['design']['rows']]
    validate_plan(plan)
    write(root/'binding.json', plan)
    print('Frozen sources and binaries; no workload executed. Binding SHA256=' + sha(root/'binding.json'))


if __name__ == '__main__':
    main()
