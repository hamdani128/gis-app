export const reportRoutes: Record<string, string> = {
  rlist_acq_process: "/admin/report/list-acquisition-process",
  mapreport: "/admin/report/map-acquisition-progress",
  acqprogress: "/admin/report/acquisition-progress",
  detaillandacq: "/admin/report/detail-land-acquisition",
  rpt_all_progress: "/admin/report/all-estate-progress",
  plantmaprep: "/admin/report/map-planting",
  reportdocland: "/admin/report/compensated-land-document",
  aestategrttreport: "/admin/report/estate-progress-report",
};

export function reportPath(moduleCode: string) {
  return reportRoutes[moduleCode.trim().toLowerCase()] ?? null;
}
