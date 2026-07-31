# Account Export Format

This document implements `DESIGN-008` `DataExporter` and supports `SW-REQ-043` and `SW-REQ-072`.

`GET /api/v1/account/export?format=json|csv` returns the authenticated account's portable data. Every persistence query is scoped with the server-derived authenticated user ID. Global catalog records and another user's records are not part of Account Export.

## Identity and owner-free records

The account identity appears only in the JSON `user` object or the CSV `user` section. Nested saved items, saved diets and entries, search history, consent, and custom items are export-only projections. They do not contain persistence ownership fields such as `userId`, `UserID`, `ownerId`, or `owner_id`.

Removing repeated ownership metadata does not remove portable content. Resource IDs, saved-item kinds, diet names and ordered quantities, search queries and filter hashes, custom-item metric serving and density data, macronutrients, micronutrients, classifications, images, and timestamps remain present where applicable.

## JSON

The JSON document has exactly these top-level fields:

`user`, `consent`, `savedItems`, `savedDiets`, `history`, and `customItems`.

Collections are arrays and remain present when empty. The frontend decoder rejects unknown fields, malformed values, legacy repository shapes, and ownership fields at any nested depth.

## CSV

CSV uses the columns `section,field,value`. The `user` section stores each top-level identity field directly. Each collection record stores its resource ID or stable legal-version key in `field` and its complete owner-free projection as escaped JSON in `value`. An empty collection is represented by `section,count,0`.

Consumers must use an RFC 4180-compatible CSV parser before parsing collection `value` cells as JSON. They must not split rows or quoted JSON cells manually.

For unchanged account data, repository ordering and deterministic JSON object encoding produce byte-identical JSON and CSV downloads.
