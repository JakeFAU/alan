from pathlib import Path
import json,shutil
src=Path('/tmp/alan-phase1f-baseline-run.py').read_text()
prefix=src[:src.index('before=old_hashes()')].replace("shutil.copyfile(__file__,OUT/'run-baseline.py')",'')
exec(compile(prefix,'baseline-prefix','exec'))
shutil.copyfile(__file__,OUT/'finalize-baseline.py')
identity=json.loads((OUT/'identity.json').read_text());FROZEN=Path(identity['FrozenDirectory']);before=json.loads((OUT/'old-artifacts-before.json').read_text())
(OUT/'audit-correction.md').write_text('The initial audit required integration-tagged adapter cancellation tests in default mode. Source build tags confirm they are selected only with integration. All full Go commands and seven Python controls had already passed. Finalization reuses those exact raw logs and selects the three untagged controls for default; integration/race retain every required cancellation control. No Go suite was rerun for this reporting correction.\n')
rest=src[src.index('sys.path.insert(0'):]
start=rest.index(' expected=required if')
end=rest.index('\n selected[mode]',start)
rest=rest[:start]+" expected=required if mode!='default' else required[:3]"+rest[end:]
exec(compile(rest,'baseline-finalize','exec'))
