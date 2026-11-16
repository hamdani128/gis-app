# Company land map

`GET /api/mapping/company-lands?company_id=7&selected_land_id=24187&after=0`

Requires the existing authenticated company-access guard. `company_id=0` requires
global access and resolves the selected land's actual company. A selected land
outside the requested company returns 404. Invalid identifiers return 400.

The normal response envelope contains a GeoJSON FeatureCollection in `data`, plus
`company_id`, `next_cursor`, `skipped` and `selected_land`. Feature properties
contain `land_id`, `status`, `owner`, `land_wide`, `wet_land_wide`, `dry_land_wide`,
`company_name`, `estate_name`, `village`, `block_plan` and `permit_number`.
Areas are database measurements in hectares; null means unavailable, while zero
is preserved. These fields come from relational land data, never legacy file
properties. `selected_land` contains the same metadata for the opened land.
The selected land's geometry is excluded because its existing map endpoint
supplies the foreground and retains that page's detail permissions.
Each page includes up to 250 lands, with every spatial record for each land.
Continue with `after=next_cursor` until zero. Invalid legacy spatial records are
skipped and counted; request/database failures are reported as errors.

Company resolution follows land → acquisition division → potential land → permit,
with the land owner's company as fallback. Source: `Gis_Data_Spatials`, supporting
the same schema variants and GeoJSON/WKT/text formats as the existing land map.
All statuses within the company are included. No write endpoint is introduced.

Code locations:

- `internal/model/mapping/company_lands.go`: company restriction, pagination, GeoJSON.
- `internal/controller/mapping/company_lands.go`: HTTP validation and responses.
- `internal/model/lands/compensated_land_spatial.go`: shared schema detection.
- Frontend `components/mapping/company-land-map.tsx`: fetching, colors, safe popups.
- Frontend `components/mapping/company-land-map.module.css`: legend styling.
- Frontend `app/admin/companies/land-info.tsx`: Leaflet foreground/background rendering.

The shared CompensatedMap view integrates this layer across acquisition and land
details. Red identifies the opened land; green indicates status 8 or 10; yellow
indicates other statuses. Clicking opens a scrollable popup with owner, hectare
measurements, company/estate, village, block plan, permit number and status for
both foreground and background lands. Text is rendered with DOM textContent.
It does not change
the active workflow form. The camera initially fits the selected land only.
Background loading and visibility changes preserve the camera. Coordinate editing
shows and saves only the selected geometry.
