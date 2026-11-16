# Report API

Report pages use the current GIS session and PostgreSQL data. They do not load the old PHP pages.

## Access

All paths require the login cookie, `company_id`, access to that company, and `p_view=1` for the report module code. Username `admin` can see all report modules, while company scope is still checked. Supported codes: `rlist_acq_process`, `mapreport`, `acqprogress`, `detaillandacq`, `rpt_all_progress`, `plantmaprep`, `reportdocland`, `aestategrttreport`.

## Backend form mapping

Each form has matching files under `internal/model/report/` and `internal/controller/report/`:

| Module code | Form file |
| --- | --- |
| `rlist_acq_process` | `list_acquisition_process.go` |
| `mapreport` | `map_acquisition_progress.go` |
| `acqprogress` | `acquisition_progress.go` |
| `detaillandacq` | `detail_land_acquisition.go` |
| `rpt_all_progress` | `all_estate_progress.go` |
| `plantmaprep` | `map_planting.go` |
| `reportdocland` | `compensated_land_document.go` |
| `aestategrttreport` | `estate_progress_report.go` |

`common.go` in each folder holds shared access, pagination, land detail, and query code. The form files own their entry points and any form-specific filters. Acquisition document storage and legacy URL fallback are reused through `internal/controller/acquisition/report_history_files.go`.

## Acquisition overview

`GET /api/reports/{code}?company_id=5&page=1&page_size=25&search=&status=&estate=`

- `page` defaults to 1. `page_size` defaults to 25 and is capped at 100.
- `status` empty means all statuses. `estate` matches an estate name exactly. `search` checks land ID, owner, village and estate.
- `plantmaprep` has its own potential-land listing and planting-map endpoint. `reportdocland` has its own compensated-land listing described below.
- `data` contains `rows`, `total`, `page`, `page_size`, `groups` (estate and status totals), `total_area` and `total_price`. The summaries cover all matching rows, not only the current page.

For a land visible in the selected company, the same prefix provides:

- `GET /api/reports/{code}/{id}/history`
- `GET /api/reports/{code}/{id}/map`
- `GET /api/reports/{code}/{id}/supported-documents`
- `GET /api/reports/{code}/{id}/supported-documents/{doc_id}`
- `GET /api/reports/{code}/{id}/supported-documents/{doc_id}/legacy-status`
- `GET /api/reports/{code}/{id}/memo`
- `GET /api/reports/{code}/{id}/memo/legacy-status`
- `GET /api/reports/{code}/{id}/memo/document`
- `GET /api/reports/{code}/{id}/local-files`
- `GET /api/reports/{code}/{id}/local-files/{name}`

Document and memo files use the same legacy-first and backend-folder fallback as the acquisition history modal.

## Compensated Land Document

`GET /api/reports/reportdocland?company_id=5&count_doc=0&year=2026&month=10&page=1&page_size=25&search=`

Only compensated lands (`status=10`) in the selected company are returned. Omit `count_doc` to include every document count, use `0` for lands without supported document records, or use a positive integer for an exact count. Optional `year` (1900–2100) and `month` (1–12) independently filter the land's `date_created`; this keeps lands without document records eligible for the period filter. Counts come from `Gis_Compensated_Land_Supported_Docs`; the response includes the land's division, owner, creation date, village, recorded area, wet/dry area and prices, admin and plant prices, calculated total and average price per hectare, and document count. `search` matches land ID, owner, village, division, or estate. The endpoint uses the report module privilege `reportdocland`.

## Land Acquisition Progress summary

`GET /api/reports/acqprogress/summary?company_id=5&division_id=0`

`division_id` is optional; zero returns all divisions in the selected company. The response contains company and permit information, configured acquisition sequences, and one row per division. Each division includes area, percentage of division area, price, and land count for each sequence's matching land status, plus a separate compensated-land group (`status=10`) and totals. The summary reads `Gis_Data_Spatials` for land and division areas; lands without valid polygon spatial data are excluded from stage aggregates and counted in `missing_spatial_count`. To match the legacy PHP report, division and compensated areas use `110.57² × 100` hectares per square degree, while stage areas use `110.93² × 100`. A land's price is counted once even when it has multiple spatial rows. The endpoint uses the same session, module privilege, and company-scope checks as the other Report endpoints.

## All Estate Progress

`GET /api/reports/rpt_all_progress/summary?company_id=5&start_date=2026-10-01&end_date=2026-10-31`

Company `0` returns all companies except the legacy placeholder company, subject to the existing all-company permission. A positive company ID limits the report to that accessible company. `start_date` and `end_date` are optional together, use `YYYY-MM-DD`, and filter compensated lands by inclusive GRTT date (maximum five years); omitting both includes every period. The response includes `start_date` and `end_date` and groups estates by company with permit area, HCV area, potential estate area, three status groups, and weighted average land price per GRTT year. Permit, HCV, and estate potential areas remain fixed when the period changes. Status groups follow the legacy form: compensated `8,10`, payment approval `7`, and on progress `0–6`. The area and percentages use the legacy `110.93² × 100` conversion of valid polygon square degrees. A land's price is counted once even with multiple spatial rows. Lands lacking valid spatial polygons are excluded from status area/price groups and counted in `missing_spatial_count`; yearly average price still uses their recorded wet/dry area and price, as in the legacy query. The endpoint requires `p_view` for `rpt_all_progress`.

## Map Planting

`GET /api/reports/plantmaprep?company_id=5&page=1&page_size=25&search=estate` returns paginated potential lands with potential name, estate name and address, estate manager, permit number, and note. `GET /api/reports/plantmaprep/{id}/planting-map?company_id=5` returns the selected estate's permit, potential, HCV, planting division, planting block, and acquisition-area WKT layers with their legacy area conversion. Both endpoints enforce the selected company and `p_view` for `plantmaprep`; company `0` requires all-company access.

## Estate Progress Report

`GET /api/reports/aestategrttreport/summary?company_id=5` returns the legacy estate progress layout for the selected company. Company `0` returns all accessible companies if the user has all-company access. It includes permit and HCV spatial areas, potential area per estate, status groups (`8,10` compensated; `7` payment approval; `0–6` on progress), weighted average price per GRTT year, and monthly area, total price and average price per hectare. Monthly and yearly price calculations use statuses `6,7,8,10`, `grtt_date`, recorded wet/dry area and prices, admin price, and plant price. The response is scoped by `aestategrttreport` view privilege. The page prints all companies and years, while the screen uses tabs to keep wide tables readable.

## List Acquisition Progress period

`GET /api/reports/rlist_acq_process/period?company_id=5&start_date=2026-10-01&end_date=2026-10-31`

Both dates use `YYYY-MM-DD`; omission defaults to the server's current month. The date range is inclusive and may span up to five years. The response includes the configured acquisition sequences, all divisions in company scope, approved process dates per land, GRTT dates, area and price totals. Rows are limited to 10,000 lands and return `truncated=true` when the limit is reached.

Spatial area follows the legacy report's degree-square conversion: `Area(mapdata) × 110.57² × 100` hectares. Malformed or absent WKT falls back to `land_wide` for the land row and marks `area_source="recorded"`. Division area and coverage percentage are shown only when valid division spatial data exists. Price per hectare is calculated as total price divided by total area.
