# Dashboard summary

`GET /api/dashboard/summary?company_id=0&year=2026`

Requires the existing login cookie. `company_id=0` is available only to users authorized for all companies; any other ID must be in the user's accessible companies. `year` is optional (defaults to the server's current year) and must be 2000–2100.

The response uses the standard `{ "message": "...", "data": { ... } }` envelope. `data` contains `company_count`, `estate_count`, `owner_count`, `land_count`, `active_area`, `estimated_price`, `completed_count`, `completed_area`, twelve `trend` months, `statuses`, `stages`, and a `breakdown`. A global request groups the breakdown by company; a company request groups it by estate.

KPI counts and the breakdown cover all recorded dates. The selected year affects only the monthly trend. Area is the recorded `Gis_Compensated_Lands.land_wide` in hectares. Active area and estimated price exclude canceled lands (status 99). Estimated price follows the land report formula: wet and dry land prices plus plant and administration prices. Completed lands use statuses 8 and 10. Monthly created counts use `date_created`; GRTT counts use `grtt_date` and exclude canceled lands. Stage counts reflect current status 0–7, not work items already assigned to an individual officer.
