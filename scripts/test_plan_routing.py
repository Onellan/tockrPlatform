#!/usr/bin/env python3
"""Contract tests for exact planner-to-router package signatures."""
from __future__ import annotations

import unittest
from pathlib import Path

from validate_plan_routing import validate_plan, validate_text

ROOT = Path(__file__).resolve().parents[1]


class PlanRoutingTests(unittest.TestCase):
    def test_each_package_has_one_compact_route_signature(self):
        plan = """
## WP01 - bounded change
Route: kind=routine; risk=R[UI]

## WP02 - elevated change
Route: kind=authorization; risk=E[AUTH]
"""
        self.assertEqual(validate_text(plan), [])
        self.assertEqual(validate_text(plan, "WP02"), [])

    def test_missing_duplicate_or_unknown_route_is_fail_closed(self):
        plan = """
## WP01 - bounded change
Route: kind=routine; risk=R[]
Route: kind=routine; risk=R[UNKNOWN]
## WP02 - missing route
"""
        issues = validate_text(plan)
        self.assertTrue(any("exactly one" in issue for issue in issues))
        self.assertTrue(any("unsupported risk code" in issue for issue in issues))

    def test_all_active_execution_plans_are_routeable(self):
        execution_plans = []
        for path in sorted((ROOT / "plan").glob("*.md")):
            text = path.read_text(encoding="utf-8")
            if any(line.startswith(("## WP", "### WP", "### Work package WP")) for line in text.splitlines()):
                execution_plans.append(path)
        # An empty set is valid once every execution plan has been moved to
        # completed history. The per-package contract is universal over the
        # active set; terminal status must not be made false merely to satisfy
        # this guard test.
        for path in execution_plans:
            self.assertEqual(validate_plan(path), [], str(path))

    def test_requested_package_must_be_present(self):
        self.assertTrue(any("not declared" in issue for issue in validate_text("## WP01\nRoute: kind=routine; risk=R[]\n", "WP99")))

    def test_duplicate_risk_codes_are_rejected(self):
        issues = validate_text("## WP01\nRoute: kind=routine; risk=R[UI,UI]\n")
        self.assertTrue(any("duplicate risk code" in issue for issue in issues))

    def test_router_plan_input_is_not_optional_in_the_contract(self):
        self.assertTrue(any("exactly one" in issue for issue in validate_text("### Work package WP01 - bounded\n", "WP01")))


if __name__ == "__main__":
    unittest.main()
