"""Process fixtures for one retained Store and one real, independently owned ledger."""
import contextlib
import fcntl
import gzip
import hashlib
import http.client
import json
import os
from pathlib import Path
import queue
import select
import signal
import socket
import sqlite3
import subprocess
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlsplit


def write(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + '\n')


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rowset(db, query, args=()):
    with contextlib.closing(sqlite3.connect(f'file:{db}?mode=ro', uri=True)) as conn:
        return list(conn.execute(query, args))


def count(db, table):
    return rowset(db, 'SELECT count(*) FROM ' + table)[0][0]


def snapshot(db, path):
    tmp = path.with_suffix('.tmp')
    with contextlib.closing(sqlite3.connect(f'file:{db}?mode=ro', uri=True)) as src, contextlib.closing(sqlite3.connect(tmp)) as dst:
        src.backup(dst)
    path.write_bytes(gzip.compress(tmp.read_bytes(), compresslevel=5, mtime=0))
    tmp.unlink()


def projections(db):
    return {s: json.loads(p) for s, p in rowset(db, 'SELECT subject,projection FROM runs')}


def value(proj, key):
    item = proj['Data'].get(key)
    if item is None:
        return None
    return item[{'int':'Number','string':'Text','bool':'Flag','list':'Items'}[item['Kind']]]


def wait_for(test, description, timeout=90):
    until = time.monotonic() + timeout
    while time.monotonic() < until:
        result = test()
        if result:
            return result
        time.sleep(.015)
    raise RuntimeError('unobserved predicate: ' + description)


class Packet:
    """A real HTTP handler blocked at ingress or after its real upstream reply."""
    def __init__(self, path, body):
        self.path, self.body = path, body
        self.accept = threading.Event()
        self.respond = threading.Event()
        self.upstream_done = threading.Event()
        self.drop = False
        self.result = None


class Relay:
    """Schedule or drop real packets; it cannot supply an application receipt."""
    def __init__(self, upstream, callback, address=0):
        target = urlsplit(upstream)
        self.incoming = queue.Queue()
        self.packets = []
        self.lock = threading.Lock()
        self.held = False

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def do_POST(h):
                body = h.rfile.read(int(h.headers['Content-Length']))
                packet = Packet(h.path, body)
                with self.lock:
                    self.packets.append(packet)
                if not self.held:
                    packet.accept.set()
                    packet.respond.set()
                self.incoming.put(packet)
                callback('transport-ingress', {'path':h.path,'body':body.decode()})
                if not packet.accept.wait(120):
                    return
                conn = http.client.HTTPConnection(target.hostname,target.port,timeout=110)
                try:
                    conn.request('POST',h.path,body,{'Content-Type':'application/json'})
                    response = conn.getresponse()
                    raw = response.read()
                    packet.result = (response.status, raw)
                    callback('transport-reply', {'path':h.path,'status':response.status,'body':raw.decode()})
                except (OSError,http.client.HTTPException) as error:
                    callback('upstream-error',str(error))
                finally:
                    conn.close()
                    packet.upstream_done.set()
                if not packet.respond.wait(120) or packet.drop or not packet.result:
                    h.close_connection = True
                    return
                try:
                    h.send_response(packet.result[0])
                    h.send_header('Content-Type','application/json')
                    h.send_header('Content-Length',str(len(packet.result[1])))
                    h.end_headers()
                    h.wfile.write(packet.result[1])
                except (BrokenPipeError,ConnectionResetError):
                    pass

        self.server = ThreadingHTTPServer(('127.0.0.1',address), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.url = f'http://127.0.0.1:{self.server.server_port}{target.path}'

    def next(self):
        return self.incoming.get(timeout=90)

    def close(self):
        self.held = False
        for p in self.packets:
            p.accept.set()
            p.respond.set()
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()


class Lab:
    def __init__(self, bins, out, work, world_copy=None, address=0, hold_ingress=False):
        self.bin, self.out, self.work = bins, out, work
        out.mkdir(parents=True)
        work.mkdir(parents=True)
        self.db = work/'local.sqlite'
        self.world = work/'world.sqlite'
        self.log = out/'journal.jsonl'
        if world_copy:
            self.world.write_bytes(gzip.decompress(world_copy.read_bytes()))
        self.trace, self.children = [], []
        self.lock = threading.Lock()
        self.hashes = {name:sha(p) for name,p in bins.items()}
        ep_args=['-receipt-ledger','-db',str(self.world),'-token',out.name,'-journal',str(self.log),'-response-mode','finish-accepted']
        self.barrier = None
        if hold_ingress:
            ar, aw = os.pipe()
            rr, rw = os.pipe()
            ackfd, relfd = fcntl.fcntl(aw,fcntl.F_DUPFD,12), fcntl.fcntl(rr,fcntl.F_DUPFD,12)
            cut = dict(Scenario='composition',Token=out.name,Store=str(self.world),Boundary='resource.ingress',Phase='pre-apply')
            launcher = 'import os,sys;os.dup2(int(sys.argv[1]),3);os.dup2(int(sys.argv[2]),4);os.execv(sys.argv[3],sys.argv[3:])'
            self.ep = subprocess.Popen([sys.executable,'-c',launcher,str(ackfd),str(relfd),str(bins['resource']),*ep_args],pass_fds=(ackfd,relfd),
                cwd=work,env={**os.environ,'ALAN_BARRIER':json.dumps(cut)},stdout=subprocess.PIPE,stderr=subprocess.PIPE)
            for fd in (aw,rr,ackfd,relfd):
                os.close(fd)
            self.barrier = (os.fdopen(ar,'rb'),os.fdopen(rw,'wb',buffering=0))
            self.children.append(self.ep)
            self.event('spawn',{'pid':self.ep.pid,'argv':self.ep.args,'cwd':str(work),'filter':cut,'sha256':self.hashes['resource']})
        else:
            self.ep = self.spawn('resource',ep_args)
        ready = self.line(self.ep)
        assert ready['PID']==self.ep.pid and ready['Store']==str(self.world)
        self.event('endpoint-ready',ready)
        self.relay = Relay(ready['URL'],self.event,address)
        self.url = self.relay.url

    def event(self, kind, data):
        with self.lock:
            self.trace.append({'event':kind,'at':time.monotonic_ns(),'data':data})

    def spawn(self,name,args,env=None):
        proc = subprocess.Popen([str(self.bin[name]),*args],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,
                                env={**os.environ,**(env or {})}, cwd=self.work)
        self.children.append(proc)
        self.event('spawn',{'pid':proc.pid,'cwd':str(self.work),'argv':proc.args,'env_overrides':env or {},'sha256':self.hashes[name]})
        return proc

    @staticmethod
    def line(proc):
        if not select.select([proc.stdout],[],[],90)[0]:
            raise RuntimeError('process acknowledgment timeout: '+str(proc.pid))
        line=proc.stdout.readline()
        if not line:
            raise RuntimeError('missing acknowledgment: '+proc.stderr.read().decode())
        return json.loads(line)

    def stop(self,proc,kill=False):
        if kill:
            proc.kill()
        output,error=proc.communicate(timeout=125)
        self.event('wait',{'pid':proc.pid,'exit':proc.returncode,'stdout':output.decode(),'stderr':error.decode()})
        if kill:
            assert proc.returncode == -signal.SIGKILL
        elif proc.returncode:
            raise RuntimeError(str(proc.args)+' '+(output+error).decode()[-3000:])
        return output

    def cli(self,op,*args):
        p=self.spawn('consumer',[op,'-db',str(self.db),'-endpoint',self.url,*args])
        return json.loads(self.stop(p))

    def owner(self,cut='',subject='',capacity=2,policy='success'):
        env=dict(ALAN_2D_OWNER_DB=str(self.db),ALAN_2D_ENDPOINT=self.url,ALAN_2D_CUT=cut,ALAN_2D_SUBJECT=subject,
                 ALAN_2D_CAPACITY=str(capacity),ALAN_2D_POLICY=policy)
        return self.spawn('owner',['-test.run=^TestPhase2DOwner$','-test.timeout=4m'],env)

    def ack(self,p,cut):
        msg=self.line(p)
        assert msg.get('cut')==cut and msg['pid']==p.pid,msg
        self.event('ack',msg)
        return msg['event']

    def save(self,label):
        snapshot(self.db,self.out/(label+'-local.sqlite.gz'))
        snapshot(self.world,self.out/(label+'-world.sqlite.gz'))
        self.event('backup',{'label':label,'journal_lines':len(self.log.read_bytes().splitlines())})

    def service(self,role,port='start'):
        return self.cli('service','-role',role,'-input',port)

    def run(self,**kw):
        p=self.owner(**kw)
        self.stop(p)

    def replay(self):
        copy=self.work/'offline.sqlite'
        packed=self.out/'final-local.sqlite.gz'
        copy.write_bytes(gzip.decompress(packed.read_bytes()))
        before=sha(copy),sha(self.log)
        p=self.spawn('consumer',['replay','-db',str(copy)])
        self.stop(p)
        after=sha(copy),sha(self.log)
        assert before==after
        self.event('offline-replay',{'before':before,'after':after})

    def close(self):
        for p in self.children:
            if p is not self.ep and p.poll() is None:
                p.kill()
                p.wait(timeout=10)
        if self.barrier:
            self.barrier[1].close()
            self.barrier[0].close()
        self.relay.close()
        if self.ep.poll() is None:
            self.ep.send_signal(signal.SIGTERM)
            self.stop(self.ep)
        write(self.out/'processes.json',self.trace)
