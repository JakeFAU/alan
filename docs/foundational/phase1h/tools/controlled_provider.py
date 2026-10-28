#!/usr/bin/env python3
"""Loopback-only controlled observations; never builds/adopts an application frontier."""

import argparse, json, threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def response(case, count, payload):
    if case in ("scalar-primary", "scalar-paired"):
        value = payload["value"]
        assert type(value) is int
        return {"observation": 0 if case == "scalar-paired" and value == 2 else value}
    if "snippet" in payload:
        return dict(
            answer="WAL records changes.",
            **{
                k: payload[k]
                for k in ("document", "version", "snippet", "start", "end")
            },
        )
    if case == "evidence" and count == 1:
        return dict(decision="clarification", tool="", query="", message="Clarify WAL")
    return dict(decision="lookup", tool="lookup", query="WAL", message="")


def main():
    p = argparse.ArgumentParser()
    p.add_argument(
        "--case",
        choices=(
            "scalar-primary",
            "scalar-paired",
            "evidence",
            "evidence-terminal-answer",
        ),
        required=True,
    )
    p.add_argument("--port", type=int, default=8765)
    p.add_argument("--ledger", required=True)
    args = p.parse_args()
    lock = threading.Lock()
    count = 0

    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            nonlocal count
            try:
                length = int(self.headers.get("Content-Length", "0"))
                assert 0 < length <= 1048576
                wire = json.loads(self.rfile.read(length))
                payload = wire
                if not args.case.startswith("scalar-"):
                    assert len(wire["input"]) == 1
                    payload = json.loads(wire["input"][0]["content"])
                with lock:
                    count += 1
                    result = response(args.case, count, payload)
                    record = dict(
                        sequence=count,
                        case=args.case,
                        payload=payload,
                        result=result,
                        client_request_id=self.headers.get("X-Client-Request-Id"),
                    )
                    f = open(args.ledger, "a")
                    f.write(json.dumps(record) + "\n")
                    f.close()
                body = (
                    result
                    if args.case.startswith("scalar-")
                    else dict(
                        status="completed",
                        model="controlled",
                        output=[
                            dict(
                                type="message",
                                role="assistant",
                                content=[
                                    dict(type="output_text", text=json.dumps(result))
                                ],
                            )
                        ],
                    )
                )
                raw = json.dumps(body).encode()
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)
            except Exception:
                self.send_error(400, "controlled fixture contract")

        def log_message(self, *args):
            pass

    server = ThreadingHTTPServer(("127.0.0.1", args.port), Handler)
    print(
        json.dumps(
            dict(endpoint=f"http://127.0.0.1:{server.server_port}", case=args.case)
        ),
        flush=True,
    )
    server.serve_forever()


if __name__ == "__main__":
    main()
