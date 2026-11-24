#!/usr/bin/env python3
"""Exercise the public README as a consumer of an exact Go module distribution."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import zipfile

MODULE = "github.com/JakeFAU/alan"
VERSION = "v0.1.0"
DRIVER = "github.com/mattn/go-sqlite3"
PUBLIC = "https://proxy.golang.org"


def command(args, cwd, env=None):
    result = subprocess.run(args, cwd=cwd, env=env, text=True, stdout=subprocess.PIPE)
    result.check_returncode()
    return result.stdout.strip()


def require(condition, message):
    if not condition:
        raise SystemExit(message)


def environment(root, proxy, sumdb):
    root.mkdir()
    return {
        "PATH": os.environ["PATH"],
        "HOME": os.environ["HOME"],
        "TMPDIR": tempfile.gettempdir(),
        "LC_ALL": "C",
        "CGO_ENABLED": "1",
        "GOWORK": "off",
        "GOTOOLCHAIN": "local",
        "GOENV": "off",
        "GOPATH": str(root / "gopath"),
        "GOCACHE": str(root / "build"),
        "GOMODCACHE": str(root / "modules"),
        "GOPROXY": proxy,
        "GOSUMDB": sumdb,
        "GOTELEMETRY": "off",
    }


def download(spec, directory, env):
    return json.loads(command(["go", "mod", "download", "-json", spec], directory, env))


def proxy_copy(entry, dest):
    # The Go proxy protocol escapes each uppercase ASCII letter with '!'.
    path = re.sub(r"[A-Z]", lambda match: "!" + match[0].lower(), entry["Path"])
    directory = dest / path / "@v"
    directory.mkdir(parents=True, exist_ok=True)
    for key in ["GoMod", "Zip", "Info"]:
        source = Path(entry[key])
        shutil.copyfile(source, directory / source.name)
    (directory / "list").write_text(entry["Version"] + "\n")


def candidate(repo, rev, work):
    sha = command(["git", "rev-parse", "--verify", rev + "^{commit}"], repo)
    bare = work / "candidate.git"
    command(["git", "clone", "--quiet", "--bare", "--shared", "--no-tags", str(repo), str(bare)], work)
    command(["git", "update-ref", "refs/tags/" + VERSION, sha], bare)
    env = environment(work / "packager", "direct", "off")
    env.update({
        "GIT_ALLOW_PROTOCOL": "file",
        "GIT_CONFIG_NOSYSTEM": "1",
        "GIT_CONFIG_GLOBAL": os.devnull,
        "GIT_CONFIG_COUNT": "1",
        "GIT_CONFIG_KEY_0": "url." + bare.as_uri() + ".insteadOf",
        "GIT_CONFIG_VALUE_0": "https://" + MODULE,
    })
    package = download(MODULE + "@" + VERSION, work, env)
    require(package.get("Origin", {}).get("Hash") == sha, "Go packaged a different Git commit")
    mod = command(["go", "mod", "edit", "-json", package["GoMod"]], work, env)
    driver_version, = [r["Version"] for r in json.loads(mod)["Require"] if r["Path"] == DRIVER]
    # The producer may use the ordinary Go dependency cache or download the pinned driver.
    driver = download(DRIVER + "@" + driver_version, repo, os.environ.copy())
    sums = (Path(package["Dir"]) / "go.sum").read_text().splitlines()
    require(f'{DRIVER} {driver_version} {driver["Sum"]}' in sums and
            f'{DRIVER} {driver_version}/go.mod {driver["GoModSum"]}' in sums, "driver checksum mismatch")
    proxy = work / "proxy"
    for entry in (package, driver):
        proxy_copy(entry, proxy)
    return sha, package, proxy.as_uri(), "off"


def block(readme, marker, language):
    pattern = r"<!-- " + marker + r" -->\n```" + language + r"\n(.*?)\n```"
    blocks = re.findall(pattern, readme, flags=re.DOTALL)
    require(len(blocks) == 1, "expected one README block for " + marker)
    return blocks[0]


def qualify(work, package, proxy, sumdb):
    with zipfile.ZipFile(package["Zip"]) as archive:
        names = archive.namelist()
        prefix = MODULE + "@" + VERSION + "/"
        readme = archive.read(prefix + "README.md").decode()
        require(not any(n.startswith((prefix + "docs/evidence/", prefix + "docs/qualification/"))
                        for n in names), "module ZIP includes archived evidence")
    env = environment(work / "consumer-cache", proxy, sumdb)
    home = work / "consumer"
    home.mkdir()
    commands = block(readme, "consumer-commands", "sh")
    expected = block(readme, "consumer-output", "text")
    output = command(["sh", "-eu", "-c", commands], home, env)
    require(output == expected, "README output mismatch:\n" + output)
    app = home / "ticket-routing"
    config = json.loads(command(["go", "mod", "edit", "-json"], app, env))
    require(not config.get("Replace") and config["Module"]["Path"] != MODULE, "unexpected module wiring")
    selected = json.loads(command(["go", "list", "-m", "-json", MODULE], app, env))
    require(selected["Version"] == VERSION and selected["Sum"] == package["Sum"] and
            selected["GoModSum"] == package["GoModSum"], "consumer resolved a different module")
    imports = command(["go", "list", "-f", '{{join .Imports " "}}', "."], app, env).split()
    require({p for p in imports if p.startswith(MODULE + "/")} ==
            {MODULE + "/" + p for p in ("author", "effects", "runtime", "schema")}, "unexpected ALAN imports")
    verify = command(["go", "mod", "verify"], app, env)
    docs = ["Create", "Open", "Store.Run", "Store.RunProgress", "Store.Close", "Store.AdmitInput",
            "Store.ApproveContinuation", "InspectExecution", "ExecutionSnapshot.Run", "ExecutionSnapshot.History",
            "Replay", "RunOptions", "Provision.AssetScopes", "ErrCapabilityUnavailable", "ErrAdapterOutput", "ErrIneligible"]
    for name in docs:
        help_text = command(["go", "doc", MODULE + "/runtime", name], app, env)
        require(name.split(".")[-1] in help_text, "missing public documentation for " + name)
    return {"output": output.splitlines(), "go_doc": docs, "module_verify": verify,
            "go_version": command(["go", "version"], app, env),
            "target": command(["go", "env", "GOOS", "GOARCH", "CGO_ENABLED"], app, env).splitlines()}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    modes = p.add_mutually_exclusive_group(required=True)
    modes.add_argument("--candidate", metavar="COMMIT")
    modes.add_argument("--published", metavar="CANDIDATE_JSON", type=Path)
    p.add_argument("--output", type=Path, required=True)
    args = p.parse_args()
    with tempfile.TemporaryDirectory(prefix="alan-distribution-") as tmp:
        work = Path(tmp).resolve()
        if args.candidate:
            repo = Path(command(["git", "rev-parse", "--show-toplevel"], Path.cwd())).resolve()
            sha, package, proxy, sumdb = candidate(repo, args.candidate, work)
        else:
            expected = json.loads(args.published.read_text())
            require(expected["mode"] == "candidate" and expected["version"] == VERSION,
                    "expected a passing candidate manifest for " + VERSION)
            proxy, sumdb = PUBLIC, "sum.golang.org"
            env = environment(work / "published", proxy, sumdb)
            package = download(MODULE + "@" + VERSION, work, env)
            require(package["Sum"] == expected["module_sum"] and package["GoModSum"] == expected["go_mod_sum"],
                    "published module differs from qualified candidate")
            tag = "refs/tags/" + VERSION
            refs = command(["git", "ls-remote", "https://" + MODULE, tag, tag + "^{}"], work).splitlines()
            hashes = {ref: sha for sha, ref in (line.split() for line in refs)}
            sha = hashes.get(tag + "^{}", hashes.get(tag))
            require(sha == expected["commit"], "published tag points at a different commit")
        result = qualify(work, package, proxy, sumdb)
        archive = Path(package["Zip"]).read_bytes()
        result.update({"mode": "candidate" if args.candidate else "published", "commit": sha,
                       "version": VERSION, "module_sum": package["Sum"], "go_mod_sum": package["GoModSum"],
                       "archive_bytes": len(archive), "archive_sha256": hashlib.sha256(archive).hexdigest()})
        report = json.dumps(result, indent=2) + "\n"
        args.output.write_text(report)
        print(report, end="")


if __name__ == "__main__":
    main()
