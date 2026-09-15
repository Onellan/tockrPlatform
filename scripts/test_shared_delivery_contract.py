#!/usr/bin/env python3
"""Contract tests for the Platform delivery contract."""
from __future__ import annotations

import hashlib
import json
import tempfile
import unittest
from pathlib import Path

import sys

try:
    import shared_delivery_contract as contract
except ModuleNotFoundError:  # pragma: no cover - package-style unittest invocation
    sys.path.insert(0, str(Path(__file__).resolve().parent))
    import shared_delivery_contract as contract

ROOT = Path(__file__).resolve().parents[1]

EXPECTED_PARITY_PROJECTION = {
    "authorized_effort": "high",
    "approval_required": True,
    "recommendation": "medium",
    "recommendation_state": "adopt-medium-candidate",
    "runtime_agent": "backlog_implementer",
    "routing_escalation": None,
    "risk_codes": [],
}


def load_sibling() -> object | None:
    # Platform deliberately adapts the shared contract to its own repository
    # boundary. CTRL/IMS parity is recorded in docs/technical/skill-inheritance.md,
    # not treated as a byte-identical runtime dependency.
    return None


class SharedDeliveryContractTests(unittest.TestCase):
    def setUp(self) -> None:
        self.candidate = {
            "commit": "candidate-commit",
            "branch": "main",
            "worktree": None,
            "fingerprint": "candidate-fingerprint",
            "id": "candidate-id",
        }

    def test_shared_file_and_identical_inputs_are_in_parity(self) -> None:
        local_path = ROOT / "scripts" / "shared_delivery_contract.py"
        digest_path = ROOT / "scripts" / "shared_delivery_contract.sha256"
        expected_digest = digest_path.read_text(encoding="utf-8").strip()
        self.assertRegex(expected_digest, r"^[0-9a-f]{64}$")
        self.assertEqual(hashlib.sha256(local_path.read_bytes()).hexdigest(), expected_digest)
        policy = contract.default_policy()
        local = contract.route_work(
            work_package="WP-PARITY", work_kind="routine", risk_level="R", risk_codes=[],
            policy=policy, supporting_gate=True,
        )
        sibling = load_sibling()
        if sibling is None:
            self.assertEqual(
                {key: local[key] for key in EXPECTED_PARITY_PROJECTION},
                EXPECTED_PARITY_PROJECTION,
            )
            return
        sibling_path = Path(sibling.__file__)  # type: ignore[attr-defined]
        self.assertEqual(hashlib.sha256(local_path.read_bytes()).digest(), hashlib.sha256(sibling_path.read_bytes()).digest())
        remote = sibling.route_work(  # type: ignore[attr-defined]
            work_package="WP-PARITY", work_kind="routine", risk_level="R", risk_codes=[],
            policy=sibling.default_policy(), supporting_gate=True,  # type: ignore[attr-defined]
        )
        self.assertEqual(local, remote)

    def test_malformed_policy_fails_closed(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "policy.json"
            path.write_text(json.dumps({"schema_version": "forged", "approved_rules": "not-a-list"}), encoding="utf-8")
            policy, warnings = contract.load_policy(path)
        decision = contract.route_work(
            work_package="WP-POLICY", work_kind="routine", risk_level="R", risk_codes=[],
            policy=policy, supporting_gate=True,
        )
        self.assertTrue(warnings)
        self.assertEqual(decision["authorized_effort"], "high")

    def test_direct_route_sanitizes_in_memory_policy(self) -> None:
        decision = contract.route_work(
            work_package="WP-DIRECT", work_kind="routine", risk_level="R", risk_codes=[],
            policy={"approved_rules": [{"id": "forged", "enabled": True, "risk_level": "R", "work_kinds": ["routine"], "evidence_gate_run_id": "gate"}]},
            supporting_gate=True, approved_gate_ids=["gate"],
        )
        self.assertEqual(decision["authorized_effort"], "high")
        self.assertTrue(decision["policy_warnings"])

    def test_risk_increase_escalates_before_mutation(self) -> None:
        decision = contract.route_work(
            work_package="WP-ESCALATE", work_kind="routine", risk_level="R", risk_codes=[],
            policy=contract.default_policy(), supporting_gate=True, observed_risk_codes=["DATA"],
        )
        self.assertEqual(decision["authorized_effort"], "high")
        self.assertEqual(decision["routing_escalation"], "high")
        self.assertIn("DATA", decision["risk_codes"])

    def test_candidate_drift_and_evidence_identity_are_fail_closed(self) -> None:
        other = dict(self.candidate, commit="different-commit", id="different-id")
        self.assertNotEqual(contract.evidence_id("validation-row", "check-1", self.candidate), contract.evidence_id("validation-row", "check-1", other))
        row = contract.evidence_row(name="check-1", status="PASS", candidate=self.candidate, executed=True, command=["check"])
        report = contract.evidence_report(kind="parity", candidate=self.candidate, rows=[row])
        self.assertEqual(contract.validate_evidence_report(report, self.candidate), [])
        self.assertTrue(contract.validate_evidence_report(report, other))

    def test_blocked_state_is_truthful_and_valid(self) -> None:
        row = contract.blocked_result(name="arm64", candidate=self.candidate, reason="QEMU unavailable")
        report = contract.evidence_report(kind="parity", candidate=self.candidate, rows=[row])
        self.assertEqual(report["status"], "BLOCKED / NOT RUN")
        self.assertEqual(contract.validate_evidence_report(report, self.candidate), [])
        self.assertFalse(row["executed"])

    def test_invalid_failure_states_fail_closed(self) -> None:
        with self.assertRaises(ValueError):
            contract.evidence_row(name="failed", status="FAIL", candidate=self.candidate, executed=True)
        with self.assertRaises(ValueError):
            contract.evidence_row(name="blocked", status="BLOCKED / NOT RUN", candidate=self.candidate, executed=False, failure_class="NONE")
        row = contract.evidence_row(name="failed", status="FAIL", candidate=self.candidate, executed=True, failure_class="TEST_FAIL")
        report = contract.evidence_report(kind="parity", candidate=self.candidate, rows=[row])
        row["failure_class"] = "NONE"
        self.assertTrue(contract.validate_evidence_report(report, self.candidate))

    def test_risk_vocabulary_is_canonical(self) -> None:
        self.assertEqual(
            set(contract.ALLOWED_RISK_CODES),
            {"DATA", "AUTH", "FIN", "HIST", "CONC", "OPS", "API", "UI", "PERF", "DEPLOY", "DEP", "GOV", "DOC"},
        )
        self.assertTrue({"DATA", "AUTH", "FIN", "HIST", "CONC", "OPS", "API", "PERF", "DEPLOY", "DEP", "GOV", "DOC"}.issubset(contract.MEDIUM_DISQUALIFYING_CODES))


if __name__ == "__main__":
    unittest.main()
