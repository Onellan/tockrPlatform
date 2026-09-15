# Operational and UI Planning

Read this reference when the item changes a user-visible workflow, external integration, authentication/secrets, deployment/startup/health, or container behaviour.

For UI work, specify the server-authorised route/handler, form or view-model change, accessible failure/empty/unknown states, and the desktop/mobile browser scenario that proves the interaction. Do not use client-side behaviour as authoritative validation or authorization.

For an external dependency, verify the current primary-source contract and identify the adapter/boundary, failure behaviour, credentials/secrets boundary, and disposable verification path. For deployment or startup work, name the relevant runtime/configuration surface, health/readiness behaviour, container verification, and any security scan required by the changed surface.

Do not add a staged rollout, a new integration layer, or a generic operational checklist unless the changed behaviour or deployment risk requires it.
