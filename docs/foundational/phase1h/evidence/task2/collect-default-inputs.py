#!/usr/bin/env python3
"""Record Go-selected default build/test inputs, independent of ignored sources.

Includes selected Go/C/assembly/embed source files and module manifests. System
headers/linked OS libraries are outside this source identity; toolchain settings
are recorded separately. This is not a replacement for the full source manifest.
"""
import argparse,gzip,hashlib,json,os,pathlib,subprocess
root=pathlib.Path(__file__).resolve().parents[5]
parser=argparse.ArgumentParser();parser.add_argument('--output');args=parser.parse_args()
out=pathlib.Path(args.output).resolve() if args.output else pathlib.Path(__file__).resolve().parent
out.mkdir(parents=True,exist_ok=True)
env=dict(os.environ);env.pop('OPENAI_API_KEY',None)
env.update(GOTOOLCHAIN='local',GOPROXY='off',GOSUMDB='off',GOFLAGS='-mod=readonly',GOCACHE='/tmp/alan-phase1f-go-build')
raw=subprocess.check_output(['go','list','-deps','-test','-json','./...'],cwd=root,env=env)
(out/'default-go-list.json.gz').write_bytes(gzip.compress(raw,mtime=0))
settings=json.loads(subprocess.check_output(['go','env','-json','GOOS','GOARCH','GOVERSION','GOROOT','GOMODCACHE','CGO_ENABLED','CC','CGO_CFLAGS','CGO_CPPFLAGS','CGO_CXXFLAGS','CGO_LDFLAGS','GOFLAGS'],cwd=root,env=env))
files={};packages=[];decoder=json.JSONDecoder();text=raw.decode();position=0
while position<len(text):
 while position<len(text) and text[position].isspace():position+=1
 if position==len(text):break
 package,position=decoder.raw_decode(text,position)
 if package.get('Error') or package.get('DepsErrors'):raise RuntimeError(package)
 packages.append(package['ImportPath'])
 directory=pathlib.Path(package.get('Dir',root))
 for key in ['GoFiles','CgoFiles','CFiles','CXXFiles','MFiles','HFiles','FFiles','SFiles','SwigFiles','SwigCXXFiles','SysoFiles','EmbedFiles']:
  for name in package.get(key,[]):
   path=pathlib.Path(name);path=path if path.is_absolute() else directory/path
   if not path.is_file():raise RuntimeError(str(path))
   digest=hashlib.sha256(path.read_bytes()).hexdigest()
   try:logical='repo/'+str(path.relative_to(root))
   except ValueError:
    try:logical='goroot/'+str(path.relative_to(settings['GOROOT']))
    except ValueError:
     try:logical='modcache/'+str(path.relative_to(settings['GOMODCACHE']))
     except ValueError:logical='generated/'+package['ImportPath']+'/'+digest
   files[logical]=digest
for path in [root/'go.mod',root/'go.sum',*(root/'docs/foundational/phase1h/evidence/fixtures').glob('*')]:
 if path.is_file():files['repo/'+str(path.relative_to(root))]=hashlib.sha256(path.read_bytes()).hexdigest()
identity={'packages':sorted(packages),'files':dict(sorted(files.items())),'settings':{k:v for k,v in settings.items() if k not in ['GOROOT','GOMODCACHE']}}
canonical=json.dumps(identity,sort_keys=True,separators=(',',':')).encode()
result={'sha256':hashlib.sha256(canonical).hexdigest(),'selected_input_count':len(files),'package_count':len(packages),'identity':identity,'notes':'Go-selected default build/test inputs plus root module/frozen fixture manifests; ignored integration/tool source is excluded; OS headers/libraries are not source-hashed.'}
(out/'default-build-inputs.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps({k:v for k,v in result.items() if k!='identity'}))
