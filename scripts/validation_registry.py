"""Platform-local validation profile registry.

Runtime profiles become executable as soon as the owning Slice introduces the
Go/SQLite surface. Container profiles become executable when the authorised
Dockerfile exists; environment/tool failures remain explicit and are never
converted to PASS.
"""

from dataclasses import dataclass


@dataclass(frozen=True)
class ValidationDefinition:
    id: str
    name: str
    kind: str
    prerequisites: tuple[str, ...] = ()
    surfaces: tuple[str, ...] = ("CORE",)

PROFILES = {
    "format": {"name": "Foundation document/configuration format", "kind": "foundation"},
    "unit": {"name": "Go unit tests", "kind": "runtime", "prerequisite": "go.mod"},
    "integration": {"name": "SQLite/HTTP integration tests", "kind": "runtime", "prerequisite": "go.mod"},
    "architecture": {"name": "Architecture and boundary contract", "kind": "foundation"},
    "security": {"name": "Security contract and secret scan", "kind": "foundation"},
    "migration": {"name": "SQLite migration evidence", "kind": "runtime", "prerequisite": "go.mod"},
    "race": {"name": "Go race validation", "kind": "runtime", "prerequisite": "go.mod"},
    "frontend": {"name": "Presentation contract and frontend toolchain", "kind": "foundation"},
    "build-amd64": {"name": "Linux AMD64 container build", "kind": "runtime", "prerequisite": "Dockerfile"},
    "build-arm64": {"name": "Linux ARM64 container build", "kind": "runtime", "prerequisite": "Dockerfile"},
    "quality": {"name": "Foundation quality checks", "kind": "foundation"},
    "full/local": {"name": "Complete local foundation profile", "kind": "composite"},
}

FULL_LOCAL = (
    "format",
    "architecture",
    "security",
    "migration",
    "frontend",
    "quality",
    "unit",
    "integration",
    "race",
    "build-amd64",
    "build-arm64",
)


def get_definition(profile: str) -> ValidationDefinition:
    """Return the stable definition shape used by evidence ledgers."""
    try:
        value = PROFILES[profile]
    except KeyError as error:
        raise KeyError(profile) from error
    prerequisite = value.get("prerequisite")
    return ValidationDefinition(
        id=profile,
        name=str(value["name"]),
        kind=str(value["kind"]),
        prerequisites=(str(prerequisite),) if prerequisite else (),
        surfaces=("CORE",),
    )
