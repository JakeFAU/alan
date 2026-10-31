# Store and standalone-consumer catalogue

The [JSONL index](catalogue.jsonl) covers the 80 automatic Store runs, 14 manual runs including adverse programs and the standalone consumer. Each `case` identifies its input archive, resource root, installed limits, top template, and every fixed run's original program and initializer. The `initializer` entry retains the exact typed projection including its PC, status and input map. Initializer IDs are audit SHA-256 digests over the compact serialized projections. The Store retains the projections themselves.

There are 95 cases, 42 programs, 28 templates, 29 distinct initial projections and 15 schemas. `source` in each deduplicated entry gives a contributing archive relative to this directory. A template names its entry program, initializer, caller port, request/report schema IDs and direct children in invocation instruction order. Equal templates in distinct roles appear twice. A program's `calls` gives the instruction PC, role, operation register, declared template and both schema IDs for each occurrence. Recursive expansion of those calls recovers a placement's complete ordered dependency tree. Program IDs hash original stored bytes. Schema entries include complete record definitions. Programs with no author metadata include the control operator in the cancellation fixture.

The [generator](catalogue.py) loads the original Store tables and checks each member program and initializer against its admitted template. It reconciles the consumer's [manifest](consumer-manifest.json.gz) with its [execution](consumer-execution.json.gz) and verifies the reachable closure against each Store's actual installed catalogue. Prefix and replay snapshots share their case's provision. The [runtime audit](runtime-audit.md) independently checks the actions and selected observations.

Run at the checkout root. The first command checks the committed file byte for byte. The second writes a fresh JSONL file to standard output.

```sh
python3 docs/foundational/phase1t/evidence/catalogue.py --check
python3 docs/foundational/phase1t/evidence/catalogue.py > /tmp/alan-phase1t-catalogue.jsonl
```

Index SHA-256 is `e7426364ce97e0b38268e092ae6482ba80b7aa2f540954a94fdd244803f13ac7`.
