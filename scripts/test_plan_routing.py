#!/usr/bin/env python3
"""Contract tests for exact planner-to-router package signatures."""
from __future__ import annotations

import unittest
import re
from pathlib import Path

from delivery_state import new_state, validate_state
from shared_delivery_contract import current_candidate
from validate_plan_routing import validate_plan, validate_text

ROOT = Path(__file__).resolve().parents[1]
PACKAGE_HEADING = re.compile(r"(?m)^\s{0,3}#{2,6}\s+(?:Work package\s+)?(?P<package>WP[A-Za-z0-9_-]*)\b")
COMPLETED_D8_S02_PLAN = Path("plan/completed/pd-d8-s02-platform-authority-certification.md")
ACTIVE_D8_S02_PLAN = Path("plan/pd-d8-s02-platform-authority-certification.md")


def d8_s02_plan_path() -> str:
    """Use the completed path after closeout, or active path before its move lands."""
    if (ROOT / COMPLETED_D8_S02_PLAN).exists():
        return COMPLETED_D8_S02_PLAN.as_posix()
    return ACTIVE_D8_S02_PLAN.as_posix()


def queued_active_plans() -> list[Path]:
    queue = (ROOT / "plan/incomplete.md").read_text(encoding="utf-8")
    paths = []
    for line in queue.splitlines():
        if not re.search(r"\*\*(?:Ready\s*/\s*next|In progress)\*\*", line, re.IGNORECASE):
            continue
        link = re.search(r"\]\(([^)]+\.md)\)", line)
        if link:
            paths.append((ROOT / "plan" / link.group(1)).resolve())
    return paths


def terminal_package(path: Path, package: str, section: str) -> bool:
    if re.search(r"(?i)Status.{0,24}(?:Terminal|Implemented\s*/\s*terminal)", section):
        return True
    for acceptance in (ROOT / "docs/implementation/audits").glob("*acceptance*.md"):
        content = acceptance.read_text(encoding="utf-8")
        if package in content and re.search(r"(?is)PASS.{0,180}terminal|terminal.{0,180}PASS", content):
            return True
    return False


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

    def test_selected_package_ignores_unrelated_terminal_packages(self):
        plan = """
## WP01 - terminal package
No historical route signature is required.

## WP06 - current package
Route: kind=defect; risk=E[GOV,DOC,API]
"""
        self.assertEqual(validate_text(plan, "WP06"), [])
        self.assertTrue(any("exactly one" in issue for issue in validate_text(plan)))

    def test_selected_package_rejects_missing_duplicate_and_unknown(self):
        missing = "## WP06 - current\n"
        duplicate = "## WP06 - current\nRoute: kind=defect; risk=E[GOV]\nRoute: kind=defect; risk=E[DOC]\n"
        valid = "## WP06 - current\nRoute: kind=defect; risk=E[GOV]\n"
        self.assertTrue(any("exactly one" in issue for issue in validate_text(missing, "WP06")))
        self.assertTrue(any("exactly one" in issue for issue in validate_text(duplicate, "WP06")))
        self.assertTrue(any("not declared" in issue for issue in validate_text(valid, "WP99")))

    def test_all_active_execution_plans_are_routeable(self):
        execution_plans = queued_active_plans()
        # An empty queue is valid after all active plans have been completed.
        # In that case there are no active packages whose route can be checked.
        for path in execution_plans:
            text = path.read_text(encoding="utf-8")
            headings = list(PACKAGE_HEADING.finditer(text))
            for index, heading in enumerate(headings):
                end = headings[index + 1].start() if index + 1 < len(headings) else len(text)
                section = text[heading.start():end]
                package = heading.group("package")
                if terminal_package(path, package, section):
                    continue
                self.assertEqual(validate_plan(path, package), [], f"{path} {package}")

    def test_queued_active_package_still_requires_a_route(self):
        for path in queued_active_plans():
            text = path.read_text(encoding="utf-8")
            headings = list(PACKAGE_HEADING.finditer(text))
            for index, heading in enumerate(headings):
                end = headings[index + 1].start() if index + 1 < len(headings) else len(text)
                section = text[heading.start():end]
                package = heading.group("package")
                if terminal_package(path, package, section):
                    continue
                self.assertRegex(section, r"(?m)^\s*Route:")
                self.assertTrue(any("exactly one" in issue for issue in validate_text(text.replace(re.search(r"(?m)^\s*Route:.*$", section).group(0), "", 1), package)))

    def test_delivery_state_allows_passed_terminal_packages_without_routes(self):
        path = d8_s02_plan_path()
        package_ids = [f"WP-PD8S02-0{number}" for number in range(1, 7)]
        state = new_state("PD-D8-S02-WP-PD8S02-06", path, current_candidate(ROOT), root=ROOT)
        state["packages"] = [{"id": package, "status": "pass"} for package in package_ids]
        state["routing"] = [{"package": package, "kind": "defect", "risk_level": "E", "risk_codes": ["GOV", "DOC", "API"], "authorized_effort": "high", "runtime_agent": "backlog_implementer"} for package in package_ids]
        self.assertEqual(validate_state(state, root=ROOT), [])

    def test_delivery_state_requires_route_for_current_unpassed_package(self):
        path = d8_s02_plan_path()
        package = "WP-PD8S02-02"
        state = new_state("PD-D8-S02-WP-PD8S02-06", path, current_candidate(ROOT), root=ROOT)
        state["packages"] = [{"id": package, "status": "running"}]
        state["routing"] = [{"package": package, "kind": "defect", "risk_level": "E", "risk_codes": ["GOV", "DOC", "API"], "authorized_effort": "high", "runtime_agent": "backlog_implementer"}]
        self.assertTrue(any("no unique plan Route signature" in issue for issue in validate_state(state, root=ROOT)))

    def test_requested_package_must_be_present(self):
        self.assertTrue(any("not declared" in issue for issue in validate_text("## WP01\nRoute: kind=routine; risk=R[]\n", "WP99")))

    def test_duplicate_risk_codes_are_rejected(self):
        issues = validate_text("## WP01\nRoute: kind=routine; risk=R[UI,UI]\n")
        self.assertTrue(any("duplicate risk code" in issue for issue in issues))

    def test_router_plan_input_is_not_optional_in_the_contract(self):
        self.assertTrue(any("exactly one" in issue for issue in validate_text("### Work package WP01 - bounded\n", "WP01")))


if __name__ == "__main__":
    unittest.main()
