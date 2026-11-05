#!/usr/bin/env python3
"""Freeze and execute separate P3 diagnostic rows after primary timing ends."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

sys.path.insert(0,str(Path(__file__).resolve().parent/'campaign'))
from custody import sha, inventory, write, snapshot, expected_custody

HERE=Path(__file__).resolve().parent


def freeze(binding,root):
    root.mkdir(parents=True,exist_ok=False)
    p=json.loads(binding.read_text())
    assert snapshot(p)==expected_custody(p)
    controls=[]
    for case in ('shared_waiting','distinct_waiting','shared_late_ready'):
        for pair in range(1,7):
            for variant in (('baseline','candidate') if pair%2 else ('candidate','baseline')):
                controls.append({'name':case+'-pair'+str(pair)+'-'+variant,'case':case,'pair':pair,'implementation':variant})
    tool_paths=[Path(__file__).resolve(),HERE/'control_overlay.py',HERE/'extract_task5.py',HERE/'selector_probe.py',HERE/'instrument.py',
                HERE.parent/'performance-p2/instrument/generate.py']
    (root/'frozen-tools').mkdir()
    for i,path in enumerate(tool_paths):
        shutil.copyfile(path,root/'frozen-tools'/(str(i)+'-'+path.name))
    plan={'version':'alan/performance-p3/diagnostics-v1','primary_binding':str(binding),'primary_binding_sha256':sha(binding),
          'environment':p['environment'],'tools':{str(f):sha(f) for f in tool_paths},
          'primary_custody':expected_custody(p),'source_candidate':p['variants']['B']['commit'],
          'controls':controls,'control_protocol':{'fresh_processes':36,'excluded_calls_per_process':1,'measured_calls_per_process':20,
             'iterations_changed_after_results':False,'fixture':'32 eligible runs; 4 or 32 genuine retained programs; optional r31 observation',
             'selection':'no Commit between calls; compare each readyRun to excluded call',
             'measurement':'wall ns and cumulative allocations for exactly20 calls, no instrumentation, forced GC or profile',
             'counter_scope':'separate one selector call per implementation/case; actual eligible IDs, loads, bindings and reductions',
             'inference':'bounded diagnostic only; independent baseline test selector within candidate source; not primary timing'},
          'counter_rows':[{'variant':v,'bound':n,'name':f'counter-b{n}-{v}'} for n in (1,8,128,256) for v in ('A','B')],
          'profile':{'variant':'B','bound':128,'name':'profile-b128-B','sampling_bytes':524288,'separate_from_primary':True}}
    write(root/'protocol.json',plan)


class Campaign:
    def __init__(self,root):
        self.root=root
        self.plan=json.loads((root/'protocol.json').read_text())
        self.primary=json.loads(Path(self.plan['primary_binding']).read_text())
        assert sha(self.plan['primary_binding'])==self.plan['primary_binding_sha256']
        assert {p:sha(p) for p in self.plan['tools']}==self.plan['tools']
        assert snapshot(self.primary)==self.plan['primary_custody']
        (root/'gates').mkdir(exist_ok=True)
        if (root/'built-inputs.json').exists():
            assert self.built_custody()==json.loads((root/'built-inputs.json').read_text())

    def built_custody(self):
        return {name:inventory(self.root/name) for name in ('counter-A','counter-B','controls')}

    def run(self,name,argv,cwd,extra=None):
        gate=self.root/'gates'/name;gate.mkdir()
        env=dict(self.plan['environment'],**(extra or {}))
        env['PYTHONDONTWRITEBYTECODE']='1'
        executable=shutil.which(str(argv[0]),path=env['PATH'])
        argv=list(map(str,argv))
        record={'argv':argv,'cwd':str(cwd),'environment':env,'executable':executable,
                'executable_sha256_before':sha(executable),'protocol_sha256':sha(self.root/'protocol.json'),
                'started_unix':time.time(),'host_load_before':os.getloadavg()}
        write(gate/'command.json',record)
        print('START '+name,flush=True)
        with (gate/'stdout.log').open('xb') as out,(gate/'stderr.log').open('xb') as err:
            child=subprocess.run(argv,cwd=cwd,env=env,stdout=out,stderr=err)
        write(gate/'outcome.json',{'exit':child.returncode,'ended_unix':time.time()})
        after={'executable_sha256_after':sha(executable),'stdout_sha256':sha(gate/'stdout.log'),
               'stderr_sha256':sha(gate/'stderr.log'),'host_load_after':os.getloadavg()}
        write(gate/'after.json',after)
        assert after['executable_sha256_after']==record['executable_sha256_before']
        print('EXIT '+name+' '+str(child.returncode),flush=True)
        if child.returncode:raise RuntimeError('diagnostic failed; original retained: '+name)
        return gate

    def build(self):
        for v in ('A','B'):
            identity=self.primary['variants'][v];dest=self.root/('counter-'+v)
            self.run('generate-counter-'+v,[self.primary['python'],HERE/'instrument.py','--alan',identity['source_dir'],
                     '--sqlite',self.primary['dependency_dir'],'--driver',HERE.parent/'performance-p2/driver','--out',dest],self.root)
            module=dest/'driver'
            shutil.copyfile(module/'go.mod',module/'measurement.mod');shutil.copyfile(module/'go.sum',module/'measurement.sum')
            self.run('configure-counter-'+v,['go','mod','edit','-modfile=measurement.mod',
                     '-replace=github.com/JakeFAU/alan='+str(dest/'alan'),'-replace=github.com/mattn/go-sqlite3='+str(dest/'sqlite')],module)
            before={n:inventory(dest/n) for n in ('alan','sqlite','driver')};write(dest/'build-inputs.json',before)
            self.run('build-counter-'+v,['go','build','-mod=readonly','-modfile=measurement.mod','-buildvcs=false','-o',dest/'driver.bin','.'],module)
            assert before=={n:inventory(dest/n) for n in before}
        controls=self.root/'controls'
        self.run('generate-controls',[self.primary['python'],HERE/'control_overlay.py',self.primary['variants']['B']['source_dir'],HERE/'selector_probe.py',controls],self.root)
        for mode in ('timed','counted'):
            source=controls/mode;before=inventory(source);write(controls/(mode+'-inputs.json'),before)
            self.run('build-controls-'+mode,['go','test','-c','-o',controls/(mode+'.test'),'./internal/executor'],source)
            assert before==inventory(source)
        write(self.root/'built-inputs.json',self.built_custody())
        self.finish('build')

    def counters(self):
        for row in self.plan['counter_rows']:
            dest=self.root/('counter-'+row['variant'])
            self.run(row['name'],[dest/'driver.bin','-bound',row['bound'],'-dir',self.root/(row['name']+'.store'),
                     '-sample',row['name'],'-source-label',self.primary['variants'][row['variant']]['commit'],
                     '-mode=direct','-prefix=true'],self.root)
        self.finish('counters')

    def controls(self):
        controls=self.root/'controls'
        self.run('control-counts',[controls/'counted.test','-test.run=^TestP3ProbeRemainingWork$','-test.v','-test.timeout=5m'],controls/'counted')
        for row in self.plan['controls']:
            self.run(row['name'],[controls/'timed.test','-test.run=^TestP3ControlTiming$','-test.v','-test.timeout=5m'],controls/'timed',
                     {'P3_CONTROL_CASE':row['case'],'P3_CONTROL_IMPL':row['implementation']})
        self.finish('controls')

    def profile(self):
        r=self.plan['profile'];binary=self.primary['variants']['B']['binary']
        self.run(r['name'],[binary,'-bound',r['bound'],'-dir',self.root/'profile.store','-sample',r['name'],
                 '-source-label',self.primary['variants']['B']['commit'],'-mode=direct','-prefix=true',
                 '-cpu-profile',self.root/'candidate-b128.cpu','-alloc-profile',self.root/'candidate-b128.alloc',
                 '-mem-profile-rate',r['sampling_bytes']],self.root)
        for kind,index in (('cpu','cpu'),('alloc','alloc_space'),('alloc','alloc_objects')):
            self.run('pprof-'+index,['go','tool','pprof','-top','-nodecount=40','-sample_index='+index,binary,self.root/('candidate-b128.'+kind)],self.root)
            self.run('pprof-cum-'+index,['go','tool','pprof','-top','-cum','-nodecount=40','-sample_index='+index,binary,self.root/('candidate-b128.'+kind)],self.root)
        self.finish('profile')

    def finish(self,phase):
        assert self.built_custody()==json.loads((self.root/'built-inputs.json').read_text())
        actual=snapshot(self.primary);assert actual==self.plan['primary_custody']
        write(self.root/(phase+'-custody.json'),{'equal':True,'actual':actual,'tools_after':{p:sha(p) for p in self.plan['tools']}})


def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('action',choices=('freeze','build','counters','controls','profile'));p.add_argument('root',type=Path);p.add_argument('--binding',type=Path)
    a=p.parse_args();root=a.root.resolve()
    if a.action=='freeze':freeze(a.binding.resolve(),root)
    else:getattr(Campaign(root),a.action)()


if __name__=='__main__':main()
