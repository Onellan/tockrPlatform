# API, Compatibility and Documentation Acceptance Validation

Load for `API` and/or `DOC` surfaces.

## API / compatibility

When an authoritative condition changes or preserves an HTTP/API/export/report contract:

- execute the public boundary directly;
- verify status, payload/field presence or absence, semantics and compatibility cases required by authority;
- include negative/cross-scope projections where sensitive or governed data is involved;
- do not treat implementation types or internal helper tests as sufficient public-contract evidence.

## Documentation truth

When completion/current-state documentation is part of the tested delta:

- run `python scripts/check_docs.py`;
- run any document-specific repository validator required by the changed document;
- verify completion/current-state claims against executed implementation evidence;
- fail unsupported claims even when the application behavior itself passes.

Do not make documentation a separate acceptance row unless product authority actually requires that documentation outcome; otherwise attach documentation truth as evidence/completion validation to the relevant acceptance row(s).