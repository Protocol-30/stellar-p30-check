#!/usr/bin/env python3
"""Exercise the built CLI with offline fixtures; never install fixture packages."""
import json
from pathlib import Path
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parent.parent
BINARY = ROOT / "p30check"


def scan(target, *flags, expected_exit=0):
    result = subprocess.run(
        [str(BINARY), "scan", str(target), "--format=json", *flags],
        cwd=ROOT, capture_output=True, text=True, check=False,
    )
    assert result.returncode == expected_exit, (target, result.returncode, result.stderr)
    report = json.loads(result.stdout)
    assert report["schema_version"] == "2"
    assert report["coverage"]["rules_implemented"] == 4
    assert report["limitations"]
    return report


def main():
    for fixture, count, cap88, helpers in [
        ("m2-js-modern", 0, "supported", "supported"),
        ("m2-js-maintenance", 1, "supported", "unsupported"),
        ("m2-js-legacy", 1, "unsupported", "unsupported"),
    ]:
        report = scan(ROOT / "testdata" / fixture, "--fail-on=high")
        assert report["coverage"]["complete"] is True
        assert len(report["findings"]) == count
        assert len(report["dependencies"]) == 1
        dependency = report["dependencies"][0]
        assert dependency["effective"] is True
        assert dependency["capabilities"]["cap88"] == cap88
        assert dependency["capabilities"]["w_address_helpers"] == helpers
        for finding in report["findings"]:
            assert finding["rule_id"] == "P30-DEP-001"
            assert finding["severity"] == "medium"
            assert finding["confidence"] == "high"
            assert finding["references"]

    for fixture in ["m2-go-observed", "m2-js-unresolved"]:
        report = scan(ROOT / "testdata" / fixture, "--fail-on=high")
        assert report["coverage"]["complete"] is True
        assert report["findings"] == []
        assert report["dependencies"][0]["effective"] is False
        assert report["dependency_warnings"]

    for fixture, count, severity, confidence in [
        ("m3-js-reject-w", 1, "high", "high"),
        ("m3-js-allow-w", 0, "", ""),
        ("m3-js-account-only", 0, "", ""),
        ("m3-js-negative-contexts", 0, "", ""),
        ("m3-js-scope-uncertain", 1, "medium", "medium"),
        ("m3-go-review", 1, "medium", "medium"),
        ("m3-js-sac-policy", 1, "medium", "low"),
    ]:
        target = ROOT / "testdata" / fixture
        default = scan(target)
        assert len(default["findings"]) == count
        gated = scan(target, "--fail-on=high", expected_exit=1 if severity == "high" else 0)
        assert gated["coverage"]["complete"] is True
        for finding in gated["findings"]:
            assert finding["rule_id"] == "P30-ADDR-001"
            assert finding["severity"] == severity
            assert finding["confidence"] == confidence
            assert finding["line"] > 0 and finding["evidence"] and finding["references"]

    for fixture, xdr_count, timing_count in [("m4-js-legacy", 1, 0), ("m4-js-modern", 0, 0), ("m4-go-review", 1, 1)]:
        report = scan(ROOT / "testdata" / fixture, "--fail-on=high")
        assert report["coverage"]["complete"] is True
        assert sum(f["rule_id"] == "P30-XDR-001" for f in report["findings"]) == xdr_count
        assert sum(f["rule_id"] == "P30-TS-001" for f in report["findings"]) == timing_count

    with tempfile.TemporaryDirectory(prefix="p30check-smoke-") as temporary:
        report = scan(Path(temporary) / "missing", expected_exit=2)
        assert report["coverage"]["complete"] is False
        assert report["error"]

    report = scan(ROOT / "testdata/m2-js-maintenance", "--max-file-bytes=1", expected_exit=2)
    assert report["coverage"]["complete"] is False
    assert report["findings"] == []
    print("CLI smoke passed: dependency, address, and ledger fixtures; actual high-gate exit1, missing root, and exhausted limit.")


if __name__ == "__main__":
    main()
