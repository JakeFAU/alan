# Independent heldout unseal report

Publication followed verification of the reviewed implementation freeze. All 185 frozen files match the pre-heldout source identity, the review and run-plan hashes match, configuration fields agree, and the retained executable bytes match. The corrected evaluation freeze and sealed commitment also pass. No output grading, inference, source/prompt tuning or previously frozen file edit occurred.

Publication began at **2026-10-26 16:05:31 UTC**, from `clock.curr_time`; implementation/configuration freeze was recorded at `2026-10-26T16:02:34.549965+00:00`. Exact commitments and command output are retained in `unseal.json`.

## Timing correction

The original evaluation freeze preceded application implementation. The version correction occurred during implementation but before any evaluation output. Earlier correction artifacts claiming the correction preceded implementation were too strong; they remain unedited historical records, superseded on timing by this statement.

## Publication and unchanged fields

All three heldout publications are exact byte-for-byte copies of current sealed inputs/gold/manifest. The original-v1 archive preserves all three original bytes, including the defective editorial citation version. No previously frozen file or semantic field was edited; all 185 frozen files were reverified after publication.

Current source: `35f21d7f4624358717176b805195a0ebfab03fd53400e92a93a4fc33d617c3f1`. Corrected evaluation freeze: `afee3bc1b40c8f51bebf16ede379183f736136e43656bf90334d02690e6ce024`. Current sealed manifest: `600b2804b5e14ee8c7d5bf255408a612a68c71519237e0567ad30337a853fa33`. Original sealed manifest: `11476bb1cce2cdd2c56873779df2a8e16f93602702aa10d84218c41bb161a4d8`.

| Published path | Bytes | SHA-256 |
| --- | ---: | --- |
| `heldout-inputs.json` | 935 | `57641425198d3a3b02c3d6e685701e51ea2b581fc78541bceccc752a69148dcb` |
| `heldout-gold.json` | 15187 | `5ddce62c4bc796c000f93548f50ac46e93198127c498605ad6e7a82fafe7a437` |
| `heldout-manifest.json` | 583 | `600b2804b5e14ee8c7d5bf255408a612a68c71519237e0567ad30337a853fa33` |
| `history/v1-sealed.tar.gz` | 2548 | `9773824d60dc103ec25e48fa7f944d96c77fe2ddd0181a0e0cff28ee34ec4f1d` |

The archive is deterministic gzip/tar with exactly three regular, flat relative members and fixed metadata. It retains the original version defect. The unchanged verifier still expects ignored private paths; the portable command reconstructs those paths only inside a temporary copy from the published JSON and historical archive. It does not use the original scratch directories.

## Reproduce from repository root

Verify the exact frozen source, review and configuration bytes (the identity sorts path components, matching the retained freeze tool):

```sh
python3 - <<'PY'
from pathlib import Path
import hashlib,json,struct
root=Path.cwd(); doc=root/'docs/foundational/phase1f'
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
read=lambda p:json.loads(p.read_text())
f=read(doc/'pre-heldout-freeze.json')
assert sha(doc/'pre-heldout-freeze.json')=='16aff76356dbb4049bb5151d8edb574f8cafe333ac6262667e7a3029274c6d4e'
assert sha(doc/f['source_manifest'])==f['source_manifest_sha256']=='3bc0497edb37053ae6f97244e08f5bc03f4e114a9d0e9cd959298251ba1dcae4'
assert sha(doc/f['independent_implementation_review'])==f['review_sha256']=='f5f635d45b031980e7887f4895eabdd2f5495e6c4096560237fbd80bd1e23901'
assert sha(doc/'evaluation-run-plan.md')==f['configuration']['run_plan_sha256']=='c161d060c2c9bb93e3e678c0112ace71149aac1c2dab05340318081f0946df6c'
config=read(doc/'evidence/live-development-v3/configuration.json')
assert sha(doc/'evidence/live-development-v3/configuration.json')=='96058c4e0fac9e05057f955cf0e0b5efecedaa4fa608523ffd29e1a7e8e2101e'
for key,value in f['configuration'].items(): assert config[key]==value
assert config['application_source_sha256']==f['source_sha256']
assert config['binary_sha256']==f['binary_sha256']
assert sha(doc/'evaluation/freeze-manifest.json')==f['configuration']['corpus_freeze']=='afee3bc1b40c8f51bebf16ede379183f736136e43656bf90334d02690e6ce024'
manifest=read(doc/f['source_manifest'])
assert len(manifest['files'])==len(set(x['path'] for x in manifest['files']))==185
# freeze.py sorts pathlib.Path values (path components), not raw strings.
assert [Path(x['path']) for x in manifest['files']]==sorted(Path(x['path']) for x in manifest['files'])
h=hashlib.sha256()
for item in manifest['files']:
    path=Path(item['path']); assert not path.is_absolute() and '..' not in path.parts
    raw=(root/path).read_bytes(); assert len(raw)==item['bytes']
    assert hashlib.sha256(raw).hexdigest()==item['sha256'],item['path']
    name=item['path'].encode()
    h.update(struct.pack('>Q',len(name))+name+struct.pack('>Q',len(raw))+raw)
assert h.hexdigest()==manifest['source_sha256']==f['source_sha256']=='35f21d7f4624358717176b805195a0ebfab03fd53400e92a93a4fc33d617c3f1'
print(json.dumps({'ok':True,'source_files':185,'source_sha256':h.hexdigest(),'review_and_configuration_verified':True},sort_keys=True))
PY
```

Actual exit: `0`. Actual output:

```text
{"ok": true, "review_and_configuration_verified": true, "source_files": 185, "source_sha256": "35f21d7f4624358717176b805195a0ebfab03fd53400e92a93a4fc33d617c3f1"}
```

Reconstruct private locations from public artifacts in a temporary copy and run the unchanged frozen verifier:

```sh
python3 - <<'PY'
from pathlib import Path
import hashlib,json,shutil,subprocess,sys,tarfile,tempfile
public=Path('docs/foundational/phase1f/evaluation').resolve()
expected={'heldout-inputs.json': {'sha256': '57641425198d3a3b02c3d6e685701e51ea2b581fc78541bceccc752a69148dcb', 'bytes': 935}, 'heldout-gold.json': {'sha256': '5ddce62c4bc796c000f93548f50ac46e93198127c498605ad6e7a82fafe7a437', 'bytes': 15187}, 'heldout-manifest.json': {'sha256': '600b2804b5e14ee8c7d5bf255408a612a68c71519237e0567ad30337a853fa33', 'bytes': 583}, 'history/v1-sealed.tar.gz': {'sha256': '9773824d60dc103ec25e48fa7f944d96c77fe2ddd0181a0e0cff28ee34ec4f1d', 'bytes': 2548}}
for name,record in expected.items():
    raw=(public/name).read_bytes()
    assert len(raw)==record['bytes']
    assert hashlib.sha256(raw).hexdigest()==record['sha256']
with tempfile.TemporaryDirectory(prefix='phase1f-unseal-') as folder:
    root=Path(folder)
    copied=root/'docs/foundational/phase1f/evaluation'
    shutil.copytree(public,copied)
    sealed=root/'.superpowers/sdd/00-plan/sealed-evaluation'
    sealed.mkdir(parents=True)
    for name in ('inputs','gold','manifest'):
        shutil.copyfile(copied/f'heldout-{name}.json',sealed/f'{name}.json')
    original=root/'.superpowers/sdd/00-plan/evaluation-freeze-v1/sealed-evaluation'
    original.mkdir(parents=True)
    with tarfile.open(copied/'history/v1-sealed.tar.gz','r:gz') as archive:
        members=archive.getmembers()
        assert sorted(item.name for item in members)==['gold.json','inputs.json','manifest.json']
        for item in members:
            path=Path(item.name)
            assert item.isfile() and not path.is_absolute() and '..' not in path.parts
            assert len(path.parts)==1
            (original/path).write_bytes(archive.extractfile(item).read())
    result=subprocess.run([sys.executable,str(copied/'verify.py'),'--sealed'],cwd=root,text=True,capture_output=True)
    print(result.stdout,end='')
    print(result.stderr,end='',file=sys.stderr)
    print(json.dumps({'portable_verifier_exit_code':result.returncode,'published_files_checked':len(expected),'original_archive_members':3},sort_keys=True))
    assert result.returncode==0
PY
```

Actual exit: `0`. Actual output:

```text
{"checked_cases": 8, "documents": 8, "freeze_identity": "afee3bc1b40c8f51bebf16ede379183f736136e43656bf90334d02690e6ce024", "lexically_reachable_lines": 25, "lines": 25, "ok": true, "sealed_checked": true}
{"original_archive_members": 3, "portable_verifier_exit_code": 0, "published_files_checked": 4}
```

The locally retained executable was additionally hashed with the equivalent of `shasum -a 256 /tmp/alan-phase1f-investigate-v3` and matched `62e830b4ba79a6e4542d03ccf461a4a24afe2f18ad0b5ccc61f6af8d9c262e50`. That local executable path is not required for portable fixture reconstruction. Build-record SHA-256: `04e61032b46fc9111d4cca9be230ea9cc6b9822ab5fbcb53f591d4dfe9ec6b61`.

All publication/configuration hashes are in `unseal.json`. The report and unseal record are new append-only publication artifacts outside the original freeze. Gold may now be read for independent grading, but must not be supplied to inference. The frozen application and one-run-per-system heldout policy remain unchanged.

Publication and portable verification completed at **2026-10-26 16:07:21 UTC** (`clock.curr_time`). Both exact commands read back from `unseal.json` were reexecuted with exit 0 and reproduced their recorded stdout exactly.
