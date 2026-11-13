"""Execute one frozen public driver and retain its post-exit resource usage."""
import json
import os
import resource
import signal
import subprocess
import sys


if __name__ == '__main__':
    child = subprocess.run(sys.argv[1:])
    usage = resource.getrusage(resource.RUSAGE_CHILDREN)
    print(json.dumps({'child_exit': child.returncode, 'process_maxrss_bytes': usage.ru_maxrss}),
          file=sys.stderr, flush=True)
    if child.returncode < 0:
        sig = -child.returncode
        if sig not in (signal.SIGKILL, signal.SIGSTOP):
            signal.signal(sig, signal.SIG_DFL)
        os.kill(os.getpid(), sig)
    raise SystemExit(child.returncode)
