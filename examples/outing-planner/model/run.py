"""Capture the complete declared source set before compiling any local code."""
import hashlib
import json
import pathlib
import sys

SOURCES = ('check.py', 'evidence.py')


def execute_sources(sources, paths, launcher_source, argv):
    if set(sources) != set(SOURCES) or set(paths) != set(SOURCES):
        raise ValueError('require exact declared executable source set: check.py, evidence.py')
    if any(type(source) is not bytes for source in sources.values()):
        raise ValueError('source buffers must be captured immutable bytes')
    hashes = {name: hashlib.sha256(sources[name]).hexdigest() for name in SOURCES}
    manifest = json.dumps(hashes, sort_keys=True, separators=(',', ':')).encode()
    provenance = {
        'source_sha256': hashes['check.py'],
        'source_files': hashes,
        'source_set_sha256': hashlib.sha256(b'outing-source-set/v1\x00' + manifest).hexdigest(),
        'launcher_sha256': hashlib.sha256(launcher_source).hexdigest(),
        'command': argv,
        'isolated': bool(sys.flags.isolated),
        'bytecode_disabled': bool(sys.dont_write_bytecode),
    }
    # No filesystem import or module cache: only these captured buffers execute.
    model = {'__name__': 'outing_captured_model', '__file__': str(paths['check.py']), 'EXECUTION_PROVENANCE': provenance}
    evidence = {'__name__': 'outing_captured_evidence', '__file__': str(paths['evidence.py']), 'MODEL': model, 'EXECUTION_PROVENANCE': provenance}
    exec(compile(sources['check.py'], str(paths['check.py']), 'exec'), model)
    exec(compile(sources['evidence.py'], str(paths['evidence.py']), 'exec'), evidence)
    model['EVIDENCE_API'] = evidence
    model['main']()


def main():
    if not sys.flags.isolated or not sys.dont_write_bytecode:
        raise SystemExit('use python3 -I -B model/run.py COMMAND --output FILE')
    launcher = pathlib.Path(__file__)
    launcher_source = launcher.read_bytes()
    paths = {name: launcher.with_name(name) for name in SOURCES}
    sources = {name: paths[name].read_bytes() for name in SOURCES}
    execute_sources(sources, paths, launcher_source, [sys.executable, '-I', '-B'] + sys.argv[:])


if __name__ == '__main__':
    main()
