"""Bind non-repository inputs to the offline Go/CGO gates at the batch boundaries."""
import argparse
import gzip
import hashlib
import json
from pathlib import Path
import shutil
import subprocess

from gate import ROOT, sha


def command(*args):
    return subprocess.check_output(args, cwd=ROOT, text=True).strip()


def inventory():
    env=json.loads(command('go','env','-json','GOVERSION','GOROOT','GOMODCACHE','GOOS','GOARCH',
                           'CGO_ENABLED','CC','CXX','CGO_CFLAGS','CGO_LDFLAGS','GOTOOLCHAIN','GOPROXY','GOFLAGS'))
    go=Path(env['GOROOT'])
    module=json.loads(command('go','list','-m','-json','github.com/mattn/go-sqlite3'))
    assert env['CGO_ENABLED']=='1' and env['GOPROXY']=='off' and env['GOFLAGS']=='-mod=readonly'
    sdk=Path(command('xcrun','--show-sdk-path'))
    clang=Path(command('xcrun','--find','clang'))
    compiler_includes=Path(command(str(clang),'-print-resource-dir'))/'include'
    files={}
    for directory in (go/'bin',go/'pkg/tool',go/'pkg/include',go/'src',Path(module['Dir']),sdk/'usr/include',compiler_includes):
        for path in sorted(directory.rglob('*')):
            if path.is_file():
                files[str(path)]=sha(path)
    system={}
    for program in (env['CC'],'clang','ld','xcrun','go'):
        executable=Path(shutil.which(program)).resolve()
        system[str(executable)]=sha(executable)
    for p in [clang,Path(command('xcrun','--find','ld')),sdk/'SDKSettings.json',sdk/'usr/lib/libSystem.tbd',
              *sorted((sdk/'usr/lib/system').glob('*.tbd'))]:
        if p.is_file():
            system[str(p)]=sha(p)
    return dict(env=env,module=module,files=files,system=system,cc=command(env['CC'],'--version'),
                sdk=str(sdk),sdk_version=command('xcrun','--show-sdk-version'),os=command('uname','-srvmp'))


if __name__=='__main__':
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument('path',type=Path)
    ap.add_argument('--check',action='store_true')
    a=ap.parse_args()
    data=json.dumps(inventory(),sort_keys=True,separators=(',',':')).encode()
    if a.check:
        assert gzip.decompress(a.path.read_bytes())==data,'dependency snapshot changed'
    else:
        with a.path.open('xb') as f:
            f.write(gzip.compress(data, mtime=0))
    print('PASS dependency bytes',hashlib.sha256(data).hexdigest(),len(json.loads(data)['files']),'toolchain/module/header files')
