"""Serve the two fixed Responses fixtures for case.json on local loopback."""

import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


PROPOSALS = {
    "inspection": {
        "verdict": "supports",
        "quote": "The ventilation inspection passed on Monday.",
        "interpretation": "The inspection supports reopening on ventilation grounds.",
        "unresolved": "Does the building meet its other opening conditions?",
    },
    "access": {
        "verdict": "opposes",
        "quote": "The wheelchair ramp remains closed for repairs.",
        "interpretation": "The closed ramp weighs against reopening.",
        "unresolved": "When will accessible entry be restored?",
    },
}


class Fixture(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path != "/v1/responses":
            self.send_error(404)
            return
        try:
            length = int(self.headers["Content-Length"])
            if not 0 < length <= 32768:
                raise ValueError("invalid length")
            request = json.loads(self.rfile.read(length))
            data = json.loads(request["input"][0]["content"])
            proposal = PROPOSALS[data["document"]]
        except (KeyError, IndexError, ValueError, TypeError):
            self.send_error(400, "unsupported fixture request")
            return
        result = {
            "id": "fixture-" + data["document"],
            "model": "controlled-reading-room",
            "status": "completed",
            "output": [{"type": "message", "role": "assistant", "content": [
                {"type": "output_text", "text": json.dumps(proposal)}
            ]}],
            "usage": {"input_tokens": 30, "output_tokens": 20, "total_tokens": 50},
        }
        body = json.dumps(result).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    with ThreadingHTTPServer(("127.0.0.1", 18483), Fixture) as server:
        print("Fixture listening on http://127.0.0.1:18483/v1/responses", flush=True)
        try:
            server.serve_forever()
        except KeyboardInterrupt:
            pass
