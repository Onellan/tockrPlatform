# Local three-app demo

The local stack builds and runs Tockr Platform, TockrCTRL and TockrIMS with
separate SQLite databases. Platform seeds the shared User, Organisation,
Workspace and product-access graph first. Its versioned fixture manifest is
then imported by CTRL and IMS as explicit local-to-Platform identity mappings.
The apps do not share a database, and the fixture does not enable Platform
sign-in or change production authentication authority.

CTRL and IMS include the same three project references:

| Code | Project |
| --- | --- |
| `NW-2407` | Crusher Upgrade Phase 2 |
| `GL-2412` | Cold Storage Expansion |
| `MG-2503` | Substation Retrofit |

IMS installs the released South African Engineering & Infrastructure
Foundation Pack and adopts it to each demo project. The seed leaves the normal
governance baseline in draft; it does not fabricate an approval.

## Start on Windows with WSL

Install Docker Desktop, enable the WSL integration for the distribution you
use, and make sure the sibling repositories are at
`/mnt/c/MyGitProjects/tockrPlatform`, `tockrCTRL` and `tockrIMS`. In a WSL
terminal, run:

```bash
cd /mnt/c/MyGitProjects/tockrPlatform
./scripts/demo-stack.sh start
```

The first run builds the images, creates the Platform fixture, imports that
fixture into the other apps, and starts all three services. Later `start` runs
rebuild as needed and repeat the idempotent seed steps. The named Docker
volumes preserve demo data across restarts. Local service keys are stored in
`~/.local/state/tockr-demo/demo.env` with owner-only permissions.

## Logins

Use this administrator login in each application:

| App | URL | Email | Password |
| --- | --- | --- | --- |
| Tockr Platform | <http://localhost:8080> | `admin@example.com` | `Tockr-Demo-2026!` |
| TockrCTRL | <http://localhost:8029> | `admin@example.com` | `Tockr-Demo-2026!` |
| TockrIMS | <http://localhost:8030> | `admin@example.com` | `Tockr-Demo-2026!` |

Shared product users have the same password:

| Email | Name |
| --- | --- |
| `anika.demo@tockrctrl.local` | Anika Maseko |
| `sipho.demo@tockrctrl.local` | Sipho Dlamini |
| `maya.demo@tockrctrl.local` | Maya Naidoo |
| `daniel.demo@tockrctrl.local` | Daniel Jacobs |

## Try the seeded data

1. **Platform:** open the organisation and workspace. Review shared users,
   workspace roles, and active TockrCTRL and TockrIMS product access. Choose
   **Open TockrCTRL** or **Open TockrIMS** to carry the signed Platform identity
   and selected workspace into that app. You can also open **Review access**
   first to see the access boundary before continuing.
2. **TockrCTRL:** open Projects and select `NW-2407`. Explore its customers,
   workstreams, deliverables, tasks, timesheets and reports. The other two
   project references provide comparison examples.
3. **TockrIMS:** open Projects, select `NW-2407`, then inspect Governance and
   Framework. The Foundation Pack is adopted and its baseline is a draft for
   exploration. Repeat with `GL-2412` and `MG-2503`.

Platform owns canonical identities and product access. CTRL owns time,
customers and project controls. IMS owns project governance and Framework
data. The shared project codes connect product examples; each app keeps its
own project records.

The local stack keeps each product's local login available as a fallback. The
Platform handoff does not assign product roles or copy product data; each app
checks the signed identity and uses its own roles, project records and database.

## Check, view or stop the stack

```bash
./scripts/demo-stack.sh smoke
./scripts/demo-stack.sh status
./scripts/demo-stack.sh logs
./scripts/demo-stack.sh stop
```

`smoke` exercises the real Platform launcher forms into both products, checks
the signed audience and browser origin, rejects missing or untrusted origins
and assertion replays, verifies the local login fallback, and checks the linked
demo projects in both products. It also verifies that every shared demo user
can launch both products and checks the Platform machine assertion endpoint
used for non-browser integrations.
`stop` preserves all three Docker volumes. Run `start` again to resume. The
script does not delete volumes or reset the demo databases.
