"""Source-only launcher: hash and compile exactly one captured checker buffer."""
import hashlib
import pathlib
import sys


def execute_source(source, path, launcher_source, argv):
    provenance = {
        "source_sha256": hashlib.sha256(source).hexdigest(),
        "launcher_sha256": hashlib.sha256(launcher_source).hexdigest(),
        "command": argv,
        "isolated": bool(sys.flags.isolated),
        "bytecode_disabled": bool(sys.dont_write_bytecode),
    }
    namespace = {"__name__": "__main__", "__file__": str(path), "EXECUTION_PROVENANCE": provenance}
    exec(compile(source, str(path), "exec"), namespace)


def main():
    if not sys.flags.isolated or not sys.dont_write_bytecode:
        raise SystemExit("use python3 -I -B model/run.py enumerate --output FILE")
    launcher = pathlib.Path(__file__)
    path = launcher.with_name("check.py")
    source = path.read_bytes()
    execute_source(source, path, launcher.read_bytes(), [sys.executable, "-I", "-B"] + sys.argv[:])


if __name__ == "__main__":
    main()
