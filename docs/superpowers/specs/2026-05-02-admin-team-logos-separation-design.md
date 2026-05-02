# Admin: separate team logos from teams

**Date:** 2026-05-02  
**Status:** Approved for implementation planning  
**Scope:** Map service admin UI and HTTP routes under `backend/cmd/map` (`handlers/admin.go`, `main.go`, templates, `admin.js`). Data model `teams.TeamLogo` unchanged.

## Goals

1. **Dedicated admin experience** for team logos (grid, search, add, edit, bulk logo actions, logos import/export) separate from team roster management.
2. **First-class URLs** for logos under `/{uuid}/admin/team-logos/...`, while keeping legacy `/{uuid}/admin/teams/logos/...` routes registered on the **same handlers** for backward compatibility (no POST redirect reliance).
3. **Split import/export** on the UI: teams page offers teams-only I/E; logos page offers logos-only I/E. Combined teams+logos export/import is not promoted in the UI (existing combined handlers may remain registered for compatibility unless removed in implementation).
4. **Policy B — platform logos (`Custom == false`):** not deletable; **name** and **symbol** (`logo` column) not admin-editable; **enabled** and **protected** may be toggled. **Custom logos:** full admin lifecycle subject to existing rules (including uploads).

## Non-goals

- Changing seed file format or initialization semantics for new deployments.
- Replacing the team logo **picker** on the teams page with async loading (teams page continues to need an enabled-logo list server-side or equivalent).
- Per-logo DELETE endpoint unless it already exists; focus is **delete all custom** semantics and import/update safety.

## Information architecture

| Page | Path | Contents |
|------|------|------------|
| Teams | `GET /{uuid}/admin/teams` | Team list/cards, create team, team bulk actions, logo **picker** per team (enabled logos + random), **teams-only** import/export. |
| Team logos | `GET /{uuid}/admin/team-logos` | Logo grid, search, add logo, edit flows, bulk enable/disable logos, **delete all custom logos**, **logos-only** import/export. |

Admin primary nav: add **Team logos** (or **Logos**) linking to `.../admin/team-logos`. Update every admin template that duplicates the nav list so the link is consistent.

**Controls** page (and any other page) that references logo bulk actions or import/export must use the new paths and, where applicable, split copy (teams vs logos).

## Routing

**Canonical (new)**

- `GET /{uuid}/admin/team-logos` — HTML handler (new or factored from current teams template data for logos).
- `GET /{uuid}/admin/team-logos/export` — `AdminTeamsExportLogosHandler` (already implemented; wire in `main.go`).
- `POST /{uuid}/admin/team-logos/import` — `AdminTeamsImportLogosHandler` (already implemented; wire in `main.go`).
- `POST /{uuid}/admin/team-logos/logos` — create logo (`AdminTeamLogosPOSTHandler`).
- `POST /{uuid}/admin/team-logos/logos/{id}` — update logo (`AdminTeamLogoUpdatePOSTHandler`).
- `POST /{uuid}/admin/team-logos/logos/enable-all`, `.../disable-all`, `.../delete-all` — existing bulk handlers (same relative suffixes as under `teams/` today).

**Legacy (retain)**

- All existing `POST /{uuid}/admin/teams/logos...` routes remain on the **same** handler functions.
- **Teams page I/E:** Wire `GET /{uuid}/admin/teams/export` → `AdminTeamsExportTeamsHandler` and `POST /{uuid}/admin/teams/import` → `AdminTeamsImportTeamsHandler` (payload is `adminTeamsTransferPayload` with `teams` populated; `logos` may be empty — teams-only import already ignores empty logos). Retarget any UI that assumed combined backup to use logos page + teams page separately. Optionally keep combined export/import on alternate paths for operators; not linked from redesigned pages.

Implementation note: duplicate chi registrations pointing at shared handler functions avoids drift.

## Policy B — server-side rules

### Single-logo update (`AdminTeamLogoUpdatePOSTHandler`)

- Load target row; if `Custom == false`:
  - Reject request with **400** if `name` in body differs from stored name (clear message: platform logo identity cannot change).
  - Do not update `logo` (already not in `Updates` map today — keep that).
  - Apply `enabled` and `protected` only.
- If `Custom == true`, keep current behavior (name + enabled + protected; symbol changes only if already supported elsewhere — do not expand scope without explicit follow-up).

### Delete all logos (`AdminTeamLogosDeleteAllPOSTHandler`)

- Today deletes every `TeamLogo` for the UUID — **incorrect for Policy B**.
- New behavior: delete **only** rows with `custom = true` (and remove any associated uploaded SVG assets on disk for those slugs, matching existing upload lifecycle).
- Response message should reflect count of deleted **custom** logos only.

### Bulk enable / disable all

- **Default (approved):** continues to update **all** logos for the UUID (platform + custom) so operators can hide the entire catalog.

### Logos import (`importAdminTeamLogosFromPayload` / `AdminTeamsImportLogosHandler`)

- Match existing row (e.g. by name) as today.
- If existing row has `Custom == false`:
  - Update **only** `enabled` and `protected` from payload (and optionally `used` — **prefer** ignoring `used` in file and running `SyncLogoUsage(uuid)` after successful import for consistency with team flows).
  - Do **not** update `name`, `logo`, or `custom`.
- If payload would **create** a new row with `custom: false`: **skip** entry (increment skip counter); do not insert fake platform rows from admin JSON.
- Custom rows: preserve existing create/update behavior.

### Export

- Logos-only export unchanged semantically; include `custom` in each row so consumers distinguish platform vs custom.
- Teams-only export: use existing teams-only shape (`AdminTeamsExportTeamsHandler`); teams page links here.
- Combined export/import: not linked from redesigned teams/logos pages unless product explicitly wants a third “full backup” control later.

## UI

### Team logos page

- Platform row: show **Platform** label (not “Not custom” alone); edit UI allows **enabled** + **protected** only; name and symbol read-only.
- Custom row: existing behavior for name/symbol/upload.
- **Delete all** button label: **Delete all custom logos**; confirmation copy states platform logos are retained.

### Teams page

- Remove: `#team-logos` section, Add Logo, logo bulk actions from Team Actions, combined import/export buttons.
- Add: Import teams / Export teams targeting teams-only endpoints (wire routes if missing in `main.go`).
- Keep: `<template id="admin-teams-logo-options-template">` (or equivalent) populated with enabled logos for the picker.

## Templates and JavaScript

- New template `admin/team-logos.html` with shared head/nav/footer pattern; include modals/partials used for logo add/edit (e.g. `add-logo.html`).
- Trim `admin/teams.html` per above.
- `admin.js`: initialize logo-only behaviors only on the team-logos page (e.g. `data-admin-page="team-logos"` or presence of a dedicated root id) so teams page does not bind logo managers. Update `data-*-url` attributes on the logos template to canonical `team-logos` paths.

## Testing and verification

- **Go tests** (extend `backend/pkg/teams` or handler tests as appropriate):
  - Non-custom logo: update with changed name → error; update toggling enabled/protected → success; `name`/`logo` unchanged.
  - Delete-all: only `custom=true` rows removed; `custom=false` remain.
  - Import: existing non-custom logo does not get `name`/`logo`/`custom` overwritten; skipped fake platform creates counted.
- **Manual:** teams picker lists logos; logos page matches Policy B; legacy `teams/logos` POST still works.

## Security notes

- Import and update handlers must not allow privilege escalation by relabeling a custom logo as platform (`custom: false`) via JSON.
- Delete-all must not remove platform rows (availability and UX for re-assigning teams).

