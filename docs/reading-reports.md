# Reading a report

What each part of a `p30check scan` report means and what to do about it.
The JSON fields are listed in [architecture.md](architecture.md#report-schema-schema_version-1);
this page uses the text output.

## Header

```text
p30check 0.1.0 - Stellar Protocol 30 compatibility scan (report schema 1)
Target:     .
Checked at: 2026-10-27T17:45:51Z
Inventory:  1 manifest file(s), 2 source file(s), 2 skipped entries
Rules run:  4
```

- **Inventory** counts the files the rules could look at: manifests and
  lockfiles (`go.mod`, `package.json`, npm lockfiles) and Go, JS/TS and
  `.x` sources. If it is lower than you expect, check **Skipped**.
- **Rules run** is 4 in v0.1.0. A smaller number means a `rules-not-implemented`
  warning explains which rule is missing.

## Findings

Each finding has a **severity** and a **confidence**, and they answer
different questions:

- **Severity**: how bad it is if the finding is right.
  `high` means the code is expected to fail on Protocol 30 ledgers or
  addresses (a decode error, a panic, a rejected address); `medium` means a
  partial break or a likely one; `info` is a hint.
- **Confidence**: how direct the evidence is. `high` means the scanner saw
  the problem itself, such as an exact SDK version below the threshold or a
  regex that accepts `M` and `C` but not `W`. `medium` means a known
  pattern whose effect depends on code it did not follow. `low` means a
  guess from incomplete information, such as a version range.

Suggested order of work:

| Severity / confidence | What to do |
|-----------------------|------------|
| high / high | Fix before the upgrade. These are the only findings `--fail-on high` gates on. |
| high / medium or low | Read the code at the line; it is usually a real break, but check the case the scanner could not see (a `default` branch, a helper elsewhere, the lockfile that actually resolves the range). |
| medium / any | Fix or decide it does not apply, and write down why. |
| info / low | Review once; most are fine (for example a check that correctly accepts only `G` and `C`). |

Each finding prints:

- `P30-...` and `path:line`: the rule and where. Rule pages, with the
  situations behind each severity, are in [rules.md](rules.md).
- `evidence`: the text the rule matched.
- `fix`: the change that removes the risk. Upgrading the SDK fixes
  P30-DEP-001 and the "old SDK" cases of P30-XDR-001; it does **not** fix
  your own address checks (P30-ADDR-001) or close-time arithmetic
  (P30-TS-001).
- `ref`: the CAP, SEP, SDK changelog or upgrade guide the finding rests on.

Test files (`_test.go`, `.test.`, `.spec.`, `__tests__/`) are not checked
by the source rules, so fixtures that hold old addresses or ledgers do not
show up.

If a finding is wrong, or a break was missed, open a
[wrong or missing finding](https://github.com/Protocol-30/stellar-p30-check/issues/new/choose)
issue with a minimal reproduction. There is no ignore comment yet
([#7](https://github.com/Protocol-30/stellar-p30-check/issues/7)).

## Warnings

Warnings are not findings and never change the exit code. They mark places
the scan could not cover.

| Code | Meaning | What to do |
|------|---------|------------|
| `manifest-unparsed` | A `go.mod`, `package.json` or npm lockfile could not be read or parsed | Its SDK versions were not checked; fix the file or check the versions by hand. |
| `source-unparsed` | A Go file has a syntax error, or a source file could not be read | The source rules skipped it. |
| `lockfile-not-parsed` | A `yarn.lock` or `pnpm-lock.yaml` is present; they are not read yet | Versions fall back to `package.json` ranges at low confidence. Check the resolved version in the lockfile ([#6](https://github.com/Protocol-30/stellar-p30-check/issues/6)). |
| `no-relevant-files` | Nothing to check under the target | Check the path; the project may be in another language or under an ignored directory. |
| `rules-not-implemented`, `check-not-implemented`, `no-rules` | The build is missing rules | Not expected with a release binary. |

## Skipped

Every entry the scanner did not inspect, with the reason:

- `ignored directory`: `.git`, `node_modules`, `vendor`, `dist`, `build`,
  `out`, `coverage` and similar. Installed and vendored copies of an SDK
  are not checked here; their version comes from your manifests and
  lockfiles.
- `larger than N bytes`: raise `--max-file-bytes` if the file matters.
- `symlink not followed` and related reasons: pass `--follow-symlinks` to
  include symlinked files inside the scan root. Symlinked directories and
  links outside the root are never followed.
- `unreadable ...`: a permissions problem on the machine running the scan.

## No findings

"Findings (0)" means none of the shapes in [rules.md](rules.md) were found
in the files listed under **Discovered files**. It is not a statement that
the project works on Protocol 30: logic split across files, other
languages, and runtime behaviour are not checked. See
[limitations.md](limitations.md).
