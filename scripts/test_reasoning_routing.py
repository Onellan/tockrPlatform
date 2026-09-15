#!/usr/bin/env python3
"""Cross-platform contract tests for evidence-gated implementer routing."""
from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ROUTER = ROOT / "scripts" / "recommend_implementer_reasoning.py"


class ReasoningRoutingTests(unittest.TestCase):
    def setUp(self) -> None:
        self.directory = tempfile.TemporaryDirectory()
        self.root = Path(self.directory.name)
        self.policy = self.root / "policy.json"
        self.report = self.root / "reports"
        self.report.mkdir()
        self.plan = self.root / "plan.md"
        self.plan.write_text("## WP1 - bounded change\nRoute: kind=routine; risk=R[]\n", encoding="utf-8")
        gate_id = "gate-test"
        scenarios = [
            "Routine bounded behavior change",
            "Defect correction",
            "Authorization/privacy change",
            "User-visible workflow",
        ]
        scenario_rows = []
        for index, scenario in enumerate(scenarios):
            baseline = self.report / f"baseline-{index}.json"
            candidate = self.report / f"candidate-{index}.json"
            common = {
                "schema_version": 2,
                "scenario": scenario,
                "quality": {"outcome": "Pass", "review_verdict": "Pass", "test_verdict": "Pass"},
                "implementation": {"head": "a" * 40, "diff_sha256": "b" * 64},
            }
            baseline.write_text(json.dumps({**common, "run_id": f"baseline-{index}", "configuration": "baseline high"}), encoding="utf-8")
            candidate.write_text(json.dumps({**common, "run_id": f"candidate-{index}", "configuration": "candidate medium", "baseline_report": str(baseline)}), encoding="utf-8")
            scenario_rows.append({
                "scenario": scenario, "verdict": "Pass", "best_metric": "input_tokens",
                "best_improvement_percent": 20.0, "baseline_report": str(baseline), "candidate_report": str(candidate),
            })
        (self.report / f"agent-eval-gate-{gate_id}.json").write_text(json.dumps({
            "schema_version": 1,
            "run_id": gate_id,
            "created_at_utc": "2026-09-09T00:00:00Z",
            "baseline_configuration": "implementer high",
            "candidate_configuration": "implementer medium",
            "minimum_improvement_percent": 10.0,
            "required_scenarios_missing": [],
            "result": "Adopt medium",
            "scenario_results": scenario_rows,
        }), encoding="utf-8")
        self.gate_id = gate_id
        for effort, total in [("high", 100), ("high", 90), ("high", 110), ("medium", 70), ("medium", 75), ("medium", 65)]:
            with (self.report / "delivery-history.jsonl").open("a", encoding="utf-8") as stream:
                stream.write(json.dumps({
                    "event": "completed", "phase": "implement", "reasoning_effort": effort,
                    "work": {"package": "WP-HISTORY", "kind": "routine", "risk_level": "R", "risk_codes": []},
                    "metrics": {"total_tokens": total},
                }) + "\n")

    def tearDown(self) -> None:
        self.directory.cleanup()

    def invoke(self, kind: str = "routine", risk: str = "R", codes: str = "", rules: list[dict[str, object]] | None = None, check: bool = True, high_agent: str = "backlog_implementer", medium_agent: str = "backlog_implementer_medium", work_package: str = "WP1") -> subprocess.CompletedProcess[str] | dict[str, object]:
        self.policy.write_text(json.dumps({
            "schema_version": 1,
            "conceptual_role": "backlog_implementer",
            "default_effort": "high",
            "high_runtime_agent": high_agent,
            "medium_runtime_agent": medium_agent,
            "approved_rules": rules or [],
        }), encoding="utf-8")
        completed = subprocess.run([
            sys.executable, str(ROUTER), "--work-package", work_package, "--work-kind", kind,
            "--plan", str(self.plan),
            "--risk-level", risk, "--risk-codes", codes, "--policy-path", str(self.policy),
            "--report-root", str(self.report),
        ], check=check, capture_output=True, text=True)
        return json.loads(completed.stdout) if check else completed

    def test_medium_requires_policy_approval_and_history_is_descriptive(self):
        candidate = self.invoke()
        self.assertEqual(candidate["recommendation"], "medium")
        self.assertEqual(candidate["authorized_effort"], "high")
        self.assertTrue(candidate["approval_required"])
        self.assertEqual(candidate["history"]["high"]["samples"], 3)
        self.assertEqual(candidate["history"]["medium"]["samples"], 3)

        authorized = self.invoke(rules=[{
            "id": "routine-medium", "enabled": True, "risk_level": "R", "work_kinds": ["routine"],
            "risk_codes": [], "evidence_gate_run_id": self.gate_id,
        }])
        self.assertEqual(authorized["authorized_effort"], "medium")
        self.assertEqual(authorized["runtime_agent"], "backlog_implementer_medium")
        self.assertFalse(authorized["approval_required"])

    def test_required_material_surfaces_and_eh_always_route_high(self):
        for codes in ("DATA", "AUTH", "FIN", "HIST", "CONC", "OPS", "API", "PERF", "DEPLOY", "DEP", "GOV", "DOC"):
            completed = self.invoke(codes=codes, rules=[{
                "id": "overbroad", "enabled": True, "risk_level": "R", "work_kinds": ["routine"],
                "risk_codes": [codes], "evidence_gate_run_id": self.gate_id,
            }], check=False)
            result = json.loads(completed.stdout)
            self.assertNotEqual(completed.returncode, 0)
            self.assertEqual(result["authorized_effort"], "high", codes)
        for risk in ("E", "H"):
            completed = self.invoke(risk=risk, check=False)
            result = json.loads(completed.stdout)
            self.assertNotEqual(completed.returncode, 0)
            self.assertEqual(result["recommendation"], "high")
            self.assertEqual(result["authorized_effort"], "high")

    def test_routine_ui_can_use_medium_only_with_explicit_policy(self):
        self.plan.write_text("## WP1 - bounded UI change\nRoute: kind=routine; risk=R[UI]\n", encoding="utf-8")
        result = self.invoke(codes="UI", rules=[{
            "id": "routine-ui-medium", "enabled": True, "risk_level": "R", "work_kinds": ["routine"],
            "risk_codes": ["UI"], "evidence_gate_run_id": self.gate_id,
        }])
        self.assertEqual(result["recommendation"], "medium")
        self.assertEqual(result["authorized_effort"], "medium")

    def test_policy_evidence_is_candidate_local_and_surface_specific(self):
        stale = self.invoke(rules=[{
            "id": "stale", "enabled": True, "risk_level": "R", "work_kinds": ["routine"],
            "risk_codes": [], "evidence_gate_run_id": "missing",
        }])
        self.assertEqual(stale["recommendation"], "medium")
        self.assertEqual(stale["authorized_effort"], "high")
        completed = self.invoke(kind="ui", codes="UI", check=False)
        ui = json.loads(completed.stdout)
        self.assertNotEqual(completed.returncode, 0)
        self.assertEqual(ui["authorized_effort"], "high")

    def test_defect_repairs_never_route_medium_even_with_policy(self):
        completed = self.invoke(kind="defect", rules=[{
            "id": "defect-medium", "enabled": True, "risk_level": "R", "work_kinds": ["defect"],
            "risk_codes": [], "evidence_gate_run_id": self.gate_id,
        }], check=False)
        result = json.loads(completed.stdout)
        self.assertNotEqual(completed.returncode, 0)
        self.assertEqual(result["recommendation"], "high")
        self.assertEqual(result["authorized_effort"], "high")
        self.assertFalse(result["approval_required"])

    def test_malformed_policy_rule_fails_safe_to_high(self):
        result = self.invoke(high_agent="evil-high", medium_agent="evil-medium", rules=[{
            "id": "malformed", "enabled": True, "risk_level": "R", "work_kinds": ["routine"],
            "risk_codes": ["NOT-A-RISK"], "evidence_gate_run_id": self.gate_id,
        }])
        self.assertEqual(result["authorized_effort"], "high")
        self.assertTrue(result["policy_warnings"])

    def test_malformed_policy_types_fail_safe_to_high(self):
        result = self.invoke(rules=[{
            "id": "missing-codes", "enabled": True, "risk_level": "R", "work_kinds": ["routine"],
            "evidence_gate_run_id": self.gate_id,
        }])
        self.assertEqual(result["authorized_effort"], "high")
        self.assertTrue(result["policy_warnings"])

        result = self.invoke(rules=[{
            "id": "scalar-codes", "enabled": True, "risk_level": "R", "work_kinds": ["routine"],
            "risk_codes": {"not": "a-list"}, "evidence_gate_run_id": self.gate_id,
        }])
        self.assertEqual(result["authorized_effort"], "high")
        self.assertTrue(result["policy_warnings"])

        result = self.invoke(rules=[{
            "id": "object-kind", "enabled": True, "risk_level": "R", "work_kinds": [{}],
            "risk_codes": [], "evidence_gate_run_id": self.gate_id,
        }])
        self.assertEqual(result["authorized_effort"], "high")
        self.assertTrue(result["policy_warnings"])

    def test_policy_cannot_change_runtime_identity(self):
        result = self.invoke(high_agent="evil-high", medium_agent="evil-medium", rules=[{
            "id": "forged", "enabled": True, "risk_level": "R", "work_kinds": ["routine"],
            "risk_codes": [], "evidence_gate_run_id": self.gate_id,
        }])
        self.assertEqual(result["runtime_agent"], "backlog_implementer")

    def test_invalid_plan_signature_fails_safe_to_high(self):
        self.plan.write_text("## WP1 - missing route\n", encoding="utf-8")
        completed = self.invoke(check=False)
        result = json.loads(completed.stdout)
        self.assertNotEqual(completed.returncode, 0)
        self.assertEqual(result["authorized_effort"], "high")
        self.assertTrue(result["plan_routing_issues"])

    def test_undeclared_package_fails_closed_with_high_output(self):
        completed = self.invoke(check=False, work_package="WP99")
        result = json.loads(completed.stdout)
        self.assertNotEqual(completed.returncode, 0)
        self.assertEqual(result["authorized_effort"], "high")
        self.assertTrue(result["plan_routing_issues"])

    def test_supplied_routing_metadata_must_match_plan_signature(self):
        completed = self.invoke(codes="UI", check=False)
        result = json.loads(completed.stdout)
        self.assertNotEqual(completed.returncode, 0)
        self.assertEqual(result["authorized_effort"], "high")
        self.assertTrue(any("does not match" in issue for issue in result["plan_routing_issues"]))

    def test_runtime_guards_require_escalation_and_high_repair(self):
        medium = (ROOT / ".codex" / "agents" / "backlog-implementer-medium.toml").read_text(encoding="utf-8")
        delivery = (ROOT / ".codex" / "agents" / "backlog-delivery.toml").read_text(encoding="utf-8")
        planner = (ROOT / ".codex" / "agents" / "backlog-planner.toml").read_text(encoding="utf-8")
        plan_skill = (ROOT / ".agents" / "skills" / "plan-backlog-item" / "SKILL.md").read_text(encoding="utf-8")
        self.assertIn("routing_escalation=high", medium)
        self.assertIn("Repairs default high", delivery)
        self.assertIn("backlog_implementer_medium", delivery)
        self.assertIn("Do not repeatedly reload unchanged", delivery)
        self.assertIn("--plan <plan> --work-package <package>", delivery)
        self.assertIn("Route: kind=", planner)
        self.assertIn("Route: kind=", plan_skill)
        self.assertIn("frontend-design `UI:` signature", medium)
        self.assertIn("accessibility, responsive", medium)
        self.assertNotIn("no material DATA, AUTH, GOV, HIST, CONC, OPS, API, UI,", plan_skill)


if __name__ == "__main__":
    unittest.main()
