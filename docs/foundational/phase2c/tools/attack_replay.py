"""Apply single-field attacks to disposable copies of an authentic adopted Store."""
import argparse
import gzip
import hashlib
import json
from pathlib import Path
import sqlite3
import subprocess
import tempfile


def digest(data):
    return hashlib.sha256(data).hexdigest()


def replay(exe, path):
    p = subprocess.run([str(exe), 'replay', '--db', str(path)], capture_output=True, text=True)
    return p.returncode, p.stderr.strip()


def modify(conn, table, rowid, field, value):
    raw = conn.execute(f'SELECT descriptor FROM {table} WHERE rowid=?', (rowid,)).fetchone()[0]
    data = json.loads(raw)
    here = data
    for key in field[:-1]:
        here = here[key]
    old = here[field[-1]]
    assert old != value
    here[field[-1]] = value
    replacement = json.dumps(data, separators=(',', ':'), ensure_ascii=False).encode()
    conn.execute(f'UPDATE {table} SET descriptor=? WHERE rowid=?', (replacement, rowid))
    return old


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('verified_store', type=Path)
    parser.add_argument('cli', type=Path)
    parser.add_argument("--spent", type=Path, help="genuine spent-but-unreturned read cut")
    args = parser.parse_args()
    exe = args.cli.resolve()
    path = args.verified_store
    image = gzip.decompress(path.read_bytes()) if path.suffix=='.gz' else path.read_bytes()
    with tempfile.TemporaryDirectory(prefix='alan-proof-attacks-') as directory:
        directory = Path(directory)
        original = directory/'original.sqlite'
        original.write_bytes(image)
        ok, error = replay(exe,original)
        assert ok==0,error
        with sqlite3.connect(original) as conn:
            rowid, query = conn.execute('SELECT rowid,intent_id FROM receipt_read_ordinals ORDER BY ordinal DESC LIMIT 1').fetchone()
            original_row = conn.execute("SELECT rowid FROM adapter_results WHERE json_type(descriptor,'$.Reconciled')='object'").fetchone()[0]
            query_row = conn.execute('SELECT rowid FROM adapter_results WHERE intent_id=?',(query,)).fetchone()[0]
        print(json.dumps({'store':str(path),'store_image_sha256':digest(image),'replay_binary_sha256':digest(exe.read_bytes()),'baseline_exit':ok}))
        attacks = [
            ('receipt_read_ordinals',rowid,'OriginalRun.Subject','foreign'),
            ('receipt_read_ordinals',rowid,'OriginalRun.Generation',99),
            ('receipt_read_ordinals',rowid,'OriginalAuthorization','ungranted'),
            ('receipt_read_ordinals',rowid,'OriginalPreparedDigest','wrong-preparation'),
            ('receipt_read_ordinals',rowid,'OriginalTerminalSequence',0),
            ('receipt_read_ordinals',rowid,'Resolver.Epoch',0),
            ('receipt_read_ordinals',rowid,'Prefix',0),
            ('receipt_read_ordinals',rowid,'Query.TargetID','other-operation'),
            ('receipt_read_ordinals',rowid,'Query.QueryKey','foreign-query-namespace'),
            ('receipt_read_ordinals',rowid,'Query.TargetKey','foreign-original-namespace'),
            ('receipt_read_ordinals',rowid,'Query.LedgerID','f'*64),
            ('receipt_read_ordinals',rowid,'Query.Wire','{}'),
            ('receipt_read_ordinals',rowid,'Query.WireSHA256','0'*64),
            ('receipt_read_ordinals',rowid,'Query.Ordinal',3),
            ('receipt_read_ordinals',rowid,'Adopt',False),
            ('adapter_results',original_row,'Reconciled.QueryID','unrelated-read'),
            ('adapter_results',query_row,'Read.Adoption','denied'),
        ]
        for i,(table, rid, field, value) in enumerate(attacks):
            copy = directory/f'corrupt-{i}.sqlite'
            copy.write_bytes(image)
            with sqlite3.connect(copy) as conn:
                previous = modify(conn,table,rid,field.split('.'),value)
            code, reason = replay(exe,copy)
            print(json.dumps({'table':table,'field':field,'previous':previous,'replacement':value,
                              'exit':code,'stderr':reason,'disposable_sha256':digest(copy.read_bytes())}))
            assert code!=0, f'accepted corruption of {table}.{field}'


        # Recompute the ordinary digest and typed decision as a coordinated
        # adversary would. Replay must still validate the reciprocal proof.
        for label, rid in (("original-proof", original_row), ("query-decision", query_row)):
            copy = directory / (label + ".sqlite")
            copy.write_bytes(image)
            with sqlite3.connect(copy) as conn:
                data = json.loads(conn.execute("SELECT descriptor FROM adapter_results WHERE rowid=?", (rid,)).fetchone()[0])
                if label == "original-proof":
                    data["Reconciled"]["QueryID"] = "nonexistent-read"
                else:
                    data["Read"]["Adoption"] = "denied"
                    data["Values"]["adoption"]["Text"] = "denied"
                    data["Values"]["adopted"]["Flag"] = False
                data["Digest"] = ""
                encoded = lambda: json.dumps(data, separators=(",", ":"), ensure_ascii=False).encode()
                data["Digest"] = digest(encoded())
                conn.execute("UPDATE adapter_results SET descriptor=? WHERE rowid=?", (encoded(), rid))
            code, reason = replay(exe, copy)
            print(json.dumps({"coordinated": label, "exit": code, "stderr": reason}))
            assert code != 0 and "adapter result digest" not in reason, (label, reason)

        if args.spent:
            source = gzip.decompress(args.spent.read_bytes())
            copy = directory / "lost-spend.sqlite"
            copy.write_bytes(source)
            good, error = replay(exe, copy)
            assert good == 0, error
            with sqlite3.connect(copy) as conn:
                rows = conn.execute("SELECT count(*) FROM receipt_read_ordinals").fetchone()[0]
                assert rows == 1
                conn.execute("DELETE FROM receipt_read_ordinals")
            code, reason = replay(exe, copy)
            print(json.dumps({"deleted": "sole unreturned physical ordinal", "exit": code, "stderr": reason,
                              "source_sha256": digest(source)}))
            assert code != 0, "erasing a lost read would refund its allowance"


if __name__ == '__main__':
    main()
