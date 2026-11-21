#!/usr/bin/env python3
"""Regression test for the published Platform foundation audit."""

from __future__ import annotations

import json
import subprocess
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class AuditCodebaseTests(unittest.TestCase):
    def test_audit_accepts_the_authoritative_slice_inventory(self) -> None:
        result = subprocess.run(
            [sys.executable, str(ROOT / "scripts/audit_codebase.py")],
            cwd=ROOT,
            text=True,
            capture_output=True,
            check=False,
        )

        self.assertEqual(
            0,
            result.returncode,
            msg=f"audit returned {result.returncode}: {result.stdout}{result.stderr}",
        )
        report = json.loads(result.stdout)
        self.assertEqual("PASS", report["domains"]["delivery"])
        self.assertEqual(28, report["slice_plan_count"])


if __name__ == "__main__":
    unittest.main()
