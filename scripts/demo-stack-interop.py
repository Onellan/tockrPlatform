#!/usr/bin/env python3
"""Exercise the shared Platform-to-product identity flow in the local demo."""

from __future__ import annotations

import http.cookiejar
import json
import sys
import urllib.error
import urllib.parse
import urllib.request
from html.parser import HTMLParser
from pathlib import Path


PLATFORM = "http://localhost:8080"
PRODUCTS = (
    {
        "key": "product.tockrctrl",
        "audience": "tockrctrl",
        "port": 8029,
        "home_markers": (
            "Dashboard",
            "Quick log and timer",
            "Cold Storage Expansion",
            "Substation Retrofit",
            "Crusher Upgrade Phase 2",
        ),
    },
    {
        "key": "product.tockrims",
        "audience": "tockrims",
        "port": 8030,
        "home_markers": (
            "Projects",
            "GL-2412",
            "MG-2503",
            "NW-2407",
            "Cold Storage Expansion",
            "Substation Retrofit",
            "Crusher Upgrade Phase 2",
        ),
    },
)


class PageFields(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.hidden: dict[str, str] = {}
        self.forms: list[dict[str, object]] = []
        self.current_form: dict[str, object] | None = None
        self.select_id: str | None = None
        self.options: dict[str, list[tuple[str, bool]]] = {}

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        values = dict(attrs)
        if tag == "form":
            self.current_form = {
                "action": values.get("action") or "",
                "method": (values.get("method") or "get").lower(),
                "hidden": {},
            }
            self.forms.append(self.current_form)
        elif tag == "input" and values.get("type") == "hidden" and values.get("name"):
            value = values.get("value") or ""
            self.hidden[values["name"]] = value
            if self.current_form is not None:
                self.current_form["hidden"][values["name"]] = value
        elif tag == "select":
            self.select_id = values.get("id")
            if self.select_id:
                self.options[self.select_id] = []
        elif tag == "option" and self.select_id:
            self.options[self.select_id].append((values.get("value") or "", "selected" in values))

    def handle_endtag(self, tag: str) -> None:
        if tag == "form":
            self.current_form = None
        elif tag == "select":
            self.select_id = None


def require(condition: bool, message: str) -> None:
    if not condition:
        raise RuntimeError(message)


def read_demo_env(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        if len(value) >= 2 and value[0] == value[-1] and value[0] in "'\"":
            value = value[1:-1]
        values[key] = value
    return values


def new_client() -> tuple[urllib.request.OpenerDirector, http.cookiejar.CookieJar]:
    cookies = http.cookiejar.CookieJar()
    return (
        urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookies)),
        cookies,
    )


def request(
    client: urllib.request.OpenerDirector,
    url: str,
    data: bytes | None = None,
    headers: dict[str, str] | None = None,
):
    request_value = urllib.request.Request(url, data=data, headers=headers or {})
    try:
        return client.open(request_value, timeout=10)
    except urllib.error.HTTPError as error:
        return error


def fields_from(body: bytes) -> PageFields:
    fields = PageFields()
    fields.feed(body.decode("utf-8", errors="replace"))
    return fields


def response_body(response) -> str:
    return response.read().decode("utf-8", errors="replace")


def post_form(
    client: urllib.request.OpenerDirector, url: str, values: dict[str, str]
):
    return request(
        client,
        url,
        urllib.parse.urlencode(values).encode(),
        {"Content-Type": "application/x-www-form-urlencoded"},
    )


def login(client: urllib.request.OpenerDirector, base: str, email: str, password: str) -> str:
    page = request(client, f"{base}/login")
    require(page.status == 200, f"login page returned {page.status} at {base}")
    form = fields_from(page.read())
    csrf_name = next((name for name in ("csrf", "csrf_token") if form.hidden.get(name)), "")
    values = {"email": email, "password": password}
    if csrf_name:
        values[csrf_name] = form.hidden[csrf_name]
    response = post_form(client, f"{base}/login", values)
    require(response.status == 200, f"login returned {response.status} at {base}")
    return response_body(response)


def logout(client: urllib.request.OpenerDirector, base: str, page_body: str) -> None:
    form = PageFields()
    form.feed(page_body)
    csrf_name = next((name for name in ("csrf", "csrf_token") if form.hidden.get(name)), "")
    require(bool(csrf_name), f"authenticated page has no logout CSRF token at {base}")
    response = post_form(
        client, f"{base}/logout", {csrf_name: form.hidden[csrf_name]}
    )
    require(
        response.status == 200 and response.geturl().endswith("/login"),
        f"logout did not return to sign-in at {base}",
    )
    response.read()


def selected_option(fields: PageFields, selector: str) -> str:
    values = fields.options.get(selector, [])
    return next((value for value, selected in values if selected), "")


def run(env_path: Path) -> None:
    env = read_demo_env(env_path)
    email = env.get("TOCKR_ADMIN_EMAIL", "admin@example.com")
    password = env.get("TOCKR_ADMIN_PASSWORD", "")
    require(bool(password), f"demo password is missing from {env_path}")

    platform_client, platform_cookies = new_client()
    platform_home = login(platform_client, PLATFORM, email, password)
    launcher = request(platform_client, f"{PLATFORM}/")
    require(launcher.status == 200, f"Platform launcher returned {launcher.status}")
    content_policy = launcher.headers.get("Content-Security-Policy", "")
    require(
        "form-action 'self' http://localhost:8029 http://localhost:8030" in content_policy
        and "*" not in content_policy,
        "Platform browser form policy does not match the configured local product targets",
    )
    launcher_body = response_body(launcher)
    launcher_fields = fields_from(launcher_body.encode())
    organisation_id = selected_option(launcher_fields, "organisation-select")
    workspace_id = selected_option(launcher_fields, "workspace-select")
    require(bool(organisation_id and workspace_id), "Platform launcher has no active demo scope")

    keys = request(platform_client, f"{PLATFORM}/.well-known/tockr-platform-assertion-keys")
    key_rows = json.loads(response_body(keys))
    require(keys.status == 200 and any(key.get("algorithm") == "Ed25519" for key in key_rows),
            "Platform assertion verification keys are unavailable")
    csrf = next((cookie.value for cookie in platform_cookies if cookie.name == "platform_csrf"), "")
    require(bool(csrf), "Platform session CSRF cookie is missing")
    platform_headers = {"X-CSRF-Token": csrf, "Content-Type": "application/json"}
    launch_forms = [
        form for form in launcher_fields.forms
        if form["method"] == "post" and str(form["action"]).endswith("/handoff")
    ]
    require(len(launch_forms) >= len(PRODUCTS), "Platform launcher is missing browser handoff forms")
    print("PASS Platform sign-in, launcher context and assertion verification key")

    for product in PRODUCTS:
        product_key = product["key"]
        audience = product["audience"]
        base = f"http://localhost:{product['port']}"
        organisation_api = f"{PLATFORM}/api/organisations/{organisation_id}"
        access_url = (
            f"{organisation_api}/product-access/{product_key}/workspaces/{workspace_id}"
        )
        access = request(platform_client, access_url)
        access_body = json.loads(response_body(access))
        require(
            access.status == 200
            and access_body.get("allowed") is True
            and access_body.get("product_key") == product_key,
            f"Platform denied seeded access to {product_key}",
        )

        issue_url = (
            f"{organisation_api}/product-assertions/{product_key}/workspaces/{workspace_id}"
        )
        wrong_audience = "tockrims" if audience == "tockrctrl" else "tockrctrl"
        wrong_issue = request(
            platform_client,
            issue_url,
            json.dumps({"audience": wrong_audience}).encode(),
            platform_headers,
        )
        require(wrong_issue.status == 201, f"Platform could not issue negative assertion for {product_key}")
        wrong_token = json.loads(response_body(wrong_issue))["assertion"]
        wrong_handoff = request(
            new_client()[0],
            f"{base}/auth/platform/session",
            json.dumps({"assertion": wrong_token}).encode(),
            {"Content-Type": "application/json"},
        )
        require(
            wrong_handoff.status == 403,
            f"{product_key} accepted an assertion for the other product audience",
        )
        wrong_handoff.read()

        launch_form = next(
            (
                form for form in launch_forms
                if str(form["action"]).endswith(f"/{product_key}/handoff")
            ),
            None,
        )
        require(launch_form is not None, f"Platform launcher omitted {product_key} handoff")
        form_fields = launch_form["hidden"]
        require(
            form_fields.get("csrf") == csrf
            and form_fields.get("organisation_id") == organisation_id
            and form_fields.get("workspace_id") == workspace_id,
            f"{product_key} launcher form is missing its protected scope",
        )
        handoff_page = post_form(
            platform_client,
            f"{PLATFORM}{launch_form['action']}",
            {
                "csrf": csrf,
                "organisation_id": organisation_id,
                "workspace_id": workspace_id,
            },
        )
        handoff_html = response_body(handoff_page)
        require(
            handoff_page.status == 200
            and handoff_page.headers.get("Cache-Control") == "no-store",
            f"Platform browser launch returned {handoff_page.status} or was cacheable for {product_key}",
        )
        outgoing = fields_from(handoff_html.encode())
        outgoing_forms = [
            form for form in outgoing.forms
            if form["method"] == "post" and str(form["action"]).startswith(base)
        ]
        require(len(outgoing_forms) == 1, f"Platform did not render one trusted {product_key} destination")
        outgoing_form = outgoing_forms[0]
        token = outgoing_form["hidden"].get("assertion")
        require(bool(token), f"Platform browser handoff omitted assertion for {product_key}")
        require(
            outgoing_form["action"] == f"{base}/auth/platform/session",
            f"Platform browser handoff target changed for {product_key}",
        )

        missing_origin = post_form(
            new_client()[0], str(outgoing_form["action"]), {"assertion": token}
        )
        require(missing_origin.status == 403, f"{product_key} accepted a browser handoff without Origin")
        missing_origin.read()
        rejected_origin = request(
            new_client()[0],
            str(outgoing_form["action"]),
            urllib.parse.urlencode({"assertion": token}).encode(),
            {"Content-Type": "application/x-www-form-urlencoded", "Origin": "http://evil.example"},
        )
        require(rejected_origin.status == 403, f"{product_key} accepted a browser handoff from an untrusted origin")
        rejected_origin.read()

        product_client, _ = new_client()
        handoff = request(
            product_client,
            str(outgoing_form["action"]),
            urllib.parse.urlencode({"assertion": token}).encode(),
            {"Content-Type": "application/x-www-form-urlencoded", "Origin": PLATFORM},
        )
        page = response_body(handoff)
        require(
            handoff.status == 200 and handoff.geturl() == f"{base}/",
            f"{product_key} browser assertion handoff returned {handoff.status}",
        )
        for marker in product["home_markers"]:
            require(marker in page, f"{product_key} home page is missing demo record {marker}")

        replay = request(
            product_client,
            str(outgoing_form["action"]),
            urllib.parse.urlencode({"assertion": token}).encode(),
            {"Content-Type": "application/x-www-form-urlencoded", "Origin": PLATFORM},
        )
        require(replay.status == 401, f"{product_key} accepted a replayed browser assertion")
        replay.read()
        logout(product_client, base, page)

        local_client, _ = new_client()
        local_page = login(local_client, base, email, password)
        for marker in product["home_markers"]:
            require(marker in local_page, f"local {product_key} login is missing demo record {marker}")
        logout(local_client, base, local_page)
        print(f"PASS {product_key} access, audience rejection, browser handoff, origin and replay rejection, local login and seeded records")

    # Retain a direct API issuance check to cover the machine-to-machine contract
    # alongside the browser handoff exercised above.
    for product in PRODUCTS:
        product_key = product["key"]
        audience = product["audience"]
        base = f"http://localhost:{product['port']}"
        organisation_api = f"{PLATFORM}/api/organisations/{organisation_id}"
        issue_url = (
            f"{organisation_api}/product-assertions/{product_key}/workspaces/{workspace_id}"
        )
        issued = request(
            platform_client,
            issue_url,
            json.dumps({"audience": audience}).encode(),
            platform_headers,
        )
        require(issued.status == 201, f"Platform direct API assertion issuance returned {issued.status} for {product_key}")
        issued.read()
        print(f"PASS {product_key} Platform machine assertion issuance")
    logout(platform_client, PLATFORM, platform_home)

    manifest_path = env_path.parent / "platform-demo-fixture.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    require(manifest.get("fixture") == "tockr-local-demo-v1", "shared Platform identity fixture is missing or unexpected")
    for user in manifest.get("users", []):
        user_email = user.get("email", "")
        if not user_email or user_email == email:
            continue
        shared_client, _ = new_client()
        account_page = login(shared_client, PLATFORM, user_email, password)
        shared_launcher = request(shared_client, f"{PLATFORM}/")
        shared_fields = fields_from(response_body(shared_launcher).encode())
        require(shared_launcher.status == 200, f"Platform launcher returned {shared_launcher.status} for {user_email}")
        for product in PRODUCTS:
            product_key = product["key"]
            base = f"http://localhost:{product['port']}"
            launch_form = next(
                (
                    form for form in shared_fields.forms
                    if form["method"] == "post"
                    and form["action"] == f"/launch/{product_key}/handoff"
                ),
                None,
            )
            require(launch_form is not None, f"{user_email} has no authorized Platform handoff to {product_key}")
            protected_scope = launch_form["hidden"]
            response = post_form(
                shared_client,
                f"{PLATFORM}{launch_form['action']}",
                {
                    "csrf": protected_scope.get("csrf", ""),
                    "organisation_id": protected_scope.get("organisation_id", ""),
                    "workspace_id": protected_scope.get("workspace_id", ""),
                },
            )
            outgoing = fields_from(response_body(response).encode())
            target_form = next(
                (
                    form for form in outgoing.forms
                    if form["action"] == f"{base}/auth/platform/session"
                ),
                None,
            )
            require(response.status == 200 and target_form is not None, f"{user_email} could not launch {product_key}")
            product_client, _ = new_client()
            handoff = request(
                product_client,
                str(target_form["action"]),
                urllib.parse.urlencode({"assertion": target_form["hidden"].get("assertion", "")}).encode(),
                {"Content-Type": "application/x-www-form-urlencoded", "Origin": PLATFORM},
            )
            page = response_body(handoff)
            require(
                handoff.status == 200 and handoff.geturl() == f"{base}/",
                f"{user_email} handoff to {product_key} returned {handoff.status}",
            )
            for marker in product["home_markers"]:
                require(marker in page, f"{user_email} {product_key} is missing demo record {marker}")
            logout(product_client, base, page)
        logout(shared_client, PLATFORM, account_page)
    print("PASS all shared demo users hand off from Platform to both products")


if __name__ == "__main__":
    try:
        require(len(sys.argv) == 2, "usage: demo-stack-interop.py <demo-env-file>")
        run(Path(sys.argv[1]))
    except (OSError, RuntimeError, urllib.error.URLError, json.JSONDecodeError) as error:
        print(f"FAIL demo interoperability: {error}", file=sys.stderr)
        raise SystemExit(1)
