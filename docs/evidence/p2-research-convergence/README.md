# P2 / research runtime convergence evidence

This directory starts the separate draft convergence of research runtime main
`1fdd4c9e1720bf60b344914bcbc3d0938ba8d678` and reviewed P2 delivery
`49101122c31b0d790d364831f044bb996d2649fa`. The common ancestor is
`76031c34c1d0eeff325e1a8e9ca4b6e2fa2b6315`. Both histories must remain
ancestors of the distinct integration commit. No merge to main or release is
authorized. [contract.md](contract.md) and [plan.md](plan.md) are exact copies
of the contract and plan frozen before integration.

## Canonical original P2 reconstruction index

[alan-p2-evidence-index.zip](alan-p2-evidence-index.zip) is the unchanged
714,182-byte bootstrap archive from the original P2 publication. Its SHA-256 is:

```text
2d8947b0d1ba9e70f81fac9acb6098cb975014ef9b5e4ffc73993f1cfc1e4d62
```

The archive contains the original `evidence-index.json`, standard-library Python
verifiers, bootstrap manifest, seal verification, independent packaging review,
and failed first-packaging/upload records. The inner index SHA-256 is
`5e601a0e7b21469dc20c53a0a0bd13ff9dfd9ebede8aa8045ded74c11e4eaa8a`.
[publication/provenance.json](publication/provenance.json) records each copied
input's source, size and hash. This is deliberately retained public evidence
metadata; raw profiles, caches and executable binaries are not added here.

The original index attachment and its replacement returned HTTP 404. Preserve
that evidence in the three `publication/index-download-*.json` records. The
repository copy provides the canonical replacement route once its Git blob has
been independently reviewed, pushed and downloaded with matching size/hash.
Those remote verification steps remain pending; local byte equality is not a
claim of public download success. The ZIP and its original contents have not
been repackaged or edited.

The twenty original raw ZIPs remain separate original attachments. Their exact
observed URLs are in [volumes 001–010](publication/raw-volume-links-001-010.json)
and [volumes 011–020](publication/raw-volume-links-011-020.json). Matching
historical download reports are `publication/raw-downloads-*.json`; all twenty
record successful byte/hash checks. These are retained historical observations,
not fresh availability checks. Do not replace those attachments or concatenate
ZIP files directly.

## Verify and extract using a fresh directory

1. Download this canonical index ZIP and all twenty named raw ZIP attachments
   into a new directory, preserving their original names. Verify the index size
   and SHA-256 above before extracting it.
2. Extract only the index ZIP into that directory, for example with
   `python3 -m zipfile -e alan-p2-evidence-index.zip .`.
3. Run `python3 verify_p2_volumes.py . fresh-verification.json` there to verify
   all volume hashes and reconstruct/check the complete original stream.
4. To also extract all original evidence files into a new directory, run
   `python3 verify_p2_volumes.py . fresh-extraction-verification.json --extract ./retained-p2`.
   Both report paths and the extraction directory must be new. Never target the
   original sealed evidence directory or an earlier evidence export.

The index inventories 14,032 included files totaling 7,140,508,321 original bytes.
Budget at least 7.2 GB for extraction and about 477 MB for the downloaded volumes.
The twenty ordered volumes contain sequential pieces of one gzip-compressed tar.
The verifier checks their names, sizes and hashes, joined stream, safe regular-file
paths, every included file's bytes/hash, and exact inner/outer manifest agreement.
It does not open databases. The original independent review inside the ZIP
qualifies local packaging only; the original 2,818 cache and 30 compiled-executable
exclusions remain explicit in the index.

## Source and qualification boundaries

The ZIP concerns original P2 PR [#11](https://github.com/JakeFAU/alan/pull/11):
measured production implementation `2fe47cb75f57bf3f7f79779dc1841c9b9b8f28d0`,
frozen delivery `49101122c31b0d790d364831f044bb996d2649fa`, tree
`5f04fcc3f81ba78b4a53e179657dccc956cd7761`. Original P2 measurements and
qualification remain attached to those identities. Advanced main adds authority
reads, so old P2 timings, allocations and total SQL counts do not describe the
combined source. This publication is not a new performance campaign.

Original ordinary integration and competing-read repetition failures remain
material and preserved. Passing focused checks or race runs do not erase them.
The integration requires fresh combined-source qualification under the frozen
contract and independent review. Formats 13/14 continue to refuse continuations,
reconciliation and templates; convergence does not establish a combined delegated
authority and bounded-series capability. A full qualification report, remote
Git-blob verification, and separate draft convergence PR remain delivery gates.
