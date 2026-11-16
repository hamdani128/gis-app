"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import { reportPath } from "../../app/admin/report/report-routes";

type MenuItem = {
  title: string;
  href: string;
  icon: ReactNode;
};

type SidebarProps = {
  mobileOpen: boolean;
  onMobileClose: () => void;
};

type ModuleMenuItem = {
  id: number | null;
  module_name: string;
  module_code: string;
  module_file: string | null;
  menu_parent: string | null;
  sequence: number | null;
};
type AcquisitionSequenceItem = {
  id: number | null;
  name: string;
  sequence: number | null;
};
type APIEnvelope<T> = { data?: T };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

function acquisitionSlug(name: string) {
  return name.normalize("NFKD").replace(/[\u0300-\u036f]/g, "").toLowerCase()
    .replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
}

function moduleRoute(item: ModuleMenuItem) {
  const configuredRoute = (item.module_file ?? "").trim();
  if (/^\/admin\/lands\/potential-lands?$/i.test(configuredRoute)) return "/admin/lands/potentiallands";
  if (/^\/admin\/lands\/compensated-lands?$/i.test(configuredRoute)) return "/admin/lands/compensatedlands";
  if (/^\/admin\/lands\/(?:block-?plans?|div-?lands?)$/i.test(configuredRoute)) return "/admin/lands/divlands";
  if (/^\/admin\/lands\/land-?owners?$/i.test(configuredRoute)) return "/admin/lands/landowner";
  if (/^\/admin\/lands\/(?:cancell?ed-?lands?|canceledland)$/i.test(configuredRoute)) return "/admin/lands/canceledland";
  if (/^\/admin\/lands\/[a-z0-9-]+(?:\/[a-z0-9-]+)*$/i.test(configuredRoute)) return configuredRoute;
  const slug = item.module_code.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
  if (/^potential-?lands?$/.test(slug)) return "/admin/lands/potentiallands";
  if (/^compensated-?lands?$/.test(slug)) return "/admin/lands/compensatedlands";
  if (/^(?:block-?plans?|div-?lands?)$/.test(slug) || /^block plans?$/i.test(item.module_name.trim())) return "/admin/lands/divlands";
  if (/^land-?owners?$/.test(slug) || /^land owners?$/i.test(item.module_name.trim())) return "/admin/lands/landowner";
  if (/^(?:cancell?ed-?lands?|canceledland)$/.test(slug) || /^cancell?ed lands?$/i.test(item.module_name.trim())) return "/admin/lands/canceledland";
  return `/admin/lands/${slug || item.id || "module"}`;
}

const menus: MenuItem[] = [
  {
    title: "Dashboard",
    href: "/admin/dashboard",
    icon: (
      <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <rect x="2" y="2" width="9" height="9" rx="2" fill="currentColor" />
        <rect x="13" y="2" width="9" height="9" rx="2" fill="currentColor" opacity=".3" />
        <rect x="13" y="13" width="9" height="9" rx="2" fill="currentColor" opacity=".3" />
        <rect x="2" y="13" width="9" height="9" rx="2" fill="currentColor" opacity=".3" />
      </svg>
    ),
  },
  // {
  //   title: "Pengguna",
  //   href: "/admin/users",
  //   icon: (
  //     <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
  //       <path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z" fill="currentColor" />
  //       <path d="M4 21a8 8 0 0 1 16 0H4Z" fill="currentColor" opacity=".3" />
  //     </svg>
  //   ),
  // },
];

const companyMenus = [
  { title: "Region", href: "/admin/companies/region", code: "region" },
  { title: "Companies Info", href: "/admin/companies/info", code: "company" },
  { title: "Land Permits", href: "/admin/companies/land-info", code: "landpermit" },
];

const administratorMenus = [
  { title: "Ref Value Setting", href: "/admin/administrator/ref-value-setting", code: "cgrefcodes" },
  { title: "Module Setting", href: "/admin/administrator/module", code: "modul" },
  { title: "Doc Type Setting", href: "/admin/administrator/doc-type", code: "doctypesetting" },
  { title: "Officer", href: "/admin/administrator/officer", code: "aepgis_employee" },
  { title: "Business Process", href: "/admin/administrator/bussiness-process", code: "bussinessprocess" },
  { title: "Business Process Sequence", href: "/admin/administrator/bussiness-process-sequence", code: "bpseq" },
  { title: "Bussiness Process Officer", href: "/admin/administrator/bussiness-process-officer", code: "bpofficer" },
];


export default function Sidebar({ mobileOpen, onMobileClose }: SidebarProps) {
  const pathname = usePathname();
  const onAcquisitionPage = pathname.startsWith("/admin/acquisition/");
  const [administratorOpen, setAdministratorOpen] = useState(false);
  const [companiesOpen, setCompaniesOpen] = useState(false);
  const [landsOpen, setLandsOpen] = useState(() => pathname.startsWith("/admin/lands") && !onAcquisitionPage);
  const [acquisitionOpen, setAcquisitionOpen] = useState(() => onAcquisitionPage);
  const [reportOpen, setReportOpen] = useState(() => pathname.startsWith("/admin/report"));
  const [landModules, setLandModules] = useState<ModuleMenuItem[]>([]);
  const [reportModules, setReportModules] = useState<ModuleMenuItem[]>([]);
  const [acquisitionSequences, setAcquisitionSequences] = useState<AcquisitionSequenceItem[]>([]);
  const [acquisitionLoading, setAcquisitionLoading] = useState(true);
  const [visibleModules, setVisibleModules] = useState<Set<string>>(new Set());

  useEffect(() => {
    let active = true;
    const loadAccess = async () => {
      let response = await fetch(`${apiBase}/api/auth/menu-access`, { credentials: "include", cache: "no-store" });
      if (response.status === 401) {
        const renewal = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
        if (renewal.ok) response = await fetch(`${apiBase}/api/auth/menu-access`, { credentials: "include", cache: "no-store" });
      }
      if (!response.ok) throw new Error("Hak akses menu gagal dimuat.");
      const result = (await response.json()) as APIEnvelope<string[]>;
      if (active) setVisibleModules(new Set((Array.isArray(result.data) ? result.data : []).map((code) => code.trim().toLowerCase())));
    };
    void loadAccess().catch(() => { if (active) setVisibleModules(new Set()); });
    return () => { active = false; };
  }, [pathname]);

  const canView = (code: string) => visibleModules.has(code.toLowerCase());
  const visibleAdministratorMenus = administratorMenus.filter((menu) => canView(menu.code));
  const visibleCompanyMenus = companyMenus.filter((menu) => canView(menu.code));
  const visibleLandModules = landModules.filter((item) => canView(item.module_code));
  const visibleReportModules = reportModules.filter((item) => canView(item.module_code));
  const canViewAcquisitionRequest = canView("acquisitionRequest");
  const canViewAcquisitionProcesses = canView("acqProses");

  useEffect(() => {
    let active = true;
    fetch(`${apiBase}/api/modules`, { credentials: "include", cache: "no-store" })
      .then(async (response) => {
        if (!response.ok) throw new Error("Daftar modul gagal dimuat.");
        const result = (await response.json()) as APIEnvelope<ModuleMenuItem[]>;
        if (!active) return;
        const items = Array.isArray(result?.data) ? result.data : [];
        setLandModules(items
          .filter((item) => (item.menu_parent ?? "").trim().toLowerCase() === "lands" && item.module_name.trim() && item.module_code.trim())
          .sort((a, b) => (a.sequence ?? Number.MAX_SAFE_INTEGER) - (b.sequence ?? Number.MAX_SAFE_INTEGER)
            || a.module_name.localeCompare(b.module_name, undefined, { sensitivity: "base" })));
        setReportModules(items
          .filter((item) => (item.menu_parent ?? "").trim().toLowerCase() === "report" && item.module_name.trim() && item.module_code.trim() && reportPath(item.module_code))
          .sort((a, b) => (a.sequence ?? Number.MAX_SAFE_INTEGER) - (b.sequence ?? Number.MAX_SAFE_INTEGER)
            || (a.id ?? Number.MAX_SAFE_INTEGER) - (b.id ?? Number.MAX_SAFE_INTEGER)
            || a.module_name.localeCompare(b.module_name, undefined, { sensitivity: "base" })));
      })
      .catch(() => { if (active) { setLandModules([]); setReportModules([]); } });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    let active = true;
    fetch(`${apiBase}/api/business-process-sequences`, { credentials: "include", cache: "no-store" })
      .then(async (response) => {
        if (!response.ok) throw new Error("Daftar proses akuisisi gagal dimuat.");
        const result = (await response.json()) as APIEnvelope<AcquisitionSequenceItem[]>;
        if (!active) return;
        const items = Array.isArray(result?.data) ? result.data : [];
        setAcquisitionSequences(items
          .filter((item) => Number.isSafeInteger(item.id) && (item.id ?? 0) > 0
            && typeof item.name === "string" && item.name.trim()
            && !/^(?:new\s+land|acquisition\s+request)$/i.test(item.name.trim()))
          .sort((a, b) => (a.sequence ?? Number.MAX_SAFE_INTEGER) - (b.sequence ?? Number.MAX_SAFE_INTEGER)
            || (a.id ?? 0) - (b.id ?? 0)));
        setAcquisitionLoading(false);
      })
      .catch(() => { if (active) { setAcquisitionSequences([]); setAcquisitionLoading(false); } });
    return () => { active = false; };
  }, [pathname]);

  useEffect(() => {
    if (!pathname.startsWith("/admin/lands") || onAcquisitionPage) return;
    const timer = window.setTimeout(() => setLandsOpen(true), 0);
    return () => window.clearTimeout(timer);
  }, [pathname, onAcquisitionPage]);

  useEffect(() => {
    if (!onAcquisitionPage) return;
    const timer = window.setTimeout(() => setAcquisitionOpen(true), 0);
    return () => window.clearTimeout(timer);
  }, [onAcquisitionPage]);

  useEffect(() => {
    if (!pathname.startsWith("/admin/report")) return;
    const timer = window.setTimeout(() => setReportOpen(true), 0);
    return () => window.clearTimeout(timer);
  }, [pathname]);

  return (
    <aside
      id="kt_aside"
      className={`aside ${mobileOpen ? "mobile-open" : ""}`}
      data-kt-drawer="true"
      data-kt-drawer-name="aside"
      data-kt-drawer-activate="{default: true, lg: false}"
      data-kt-drawer-overlay="true"
      data-kt-drawer-width="{default:'200px', '300px': '250px'}"
      data-kt-drawer-direction="start"
    >
      <div className="aside-menu flex-column-fluid">
        <div
          className="hover-scroll-overlay-y px-2 my-5 my-lg-5"
          style={{ maxHeight: "calc(100vh - 150px)", overflowY: "auto" }}
          id="kt_aside_menu_wrapper"
          data-kt-scroll="true"
          data-kt-scroll-height="auto"
          data-kt-scroll-dependencies="#kt_aside_footer"
          data-kt-scroll-wrappers="#kt_aside_menu"
          data-kt-scroll-offset="5px"
        >
          <nav
            id="kt_aside_menu"
            aria-label="Menu utama"
            className="menu menu-column menu-title-gray-800 menu-state-title-primary menu-state-icon-primary menu-state-bullet-primary menu-arrow-gray-500"
          >
            <div className="menu-item pt-2">
              <div className="menu-content">
                <span className="menu-heading fw-bold text-uppercase fs-7">
                  Dashboard
                </span>
              </div>
            </div>

            {menus.map((menu) => {
              const active =
                pathname === menu.href || pathname.startsWith(`${menu.href}/`);

              return (
                <div className="menu-item" key={menu.href}>
                  <Link
                    href={menu.href}
                    className={`menu-link ${active ? "active" : ""}`}
                    aria-current={active ? "page" : undefined}
                    onClick={onMobileClose}
                  >
                    <span className="menu-icon">
                      <span className="svg-icon svg-icon-2">{menu.icon}</span>
                    </span>
                    <span className="menu-title">{menu.title}</span>
                  </Link>
                </div>
              );
            })}
            {(visibleAdministratorMenus.length > 0 || visibleCompanyMenus.length > 0) && <div className="menu-item pt-2">
              <div className="menu-content">
                <span className="menu-heading fw-bold text-uppercase fs-7">
                  Master
                </span>
              </div>
            </div>}
            {visibleAdministratorMenus.length > 0 && <div
              className="menu-item menu-accordion"
            >
              <button
                type="button"
                className="menu-link border-0 w-100 bg-transparent text-start"
                aria-expanded={administratorOpen}
                onClick={() => setAdministratorOpen((open) => !open)}
              >
                <span className="menu-icon">
                  <span className="svg-icon svg-icon-2">
                    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path
                        d="M12 2a2 2 0 0 1 2 2v.3a7.8 7.8 0 0 1 1.7.7l.2-.2a2 2 0 1 1 2.8 2.8l-.2.2a7.8 7.8 0 0 1 .7 1.7h.3a2 2 0 1 1 0 4h-.3a7.8 7.8 0 0 1-.7 1.7l.2.2a2 2 0 1 1-2.8 2.8l-.2-.2a7.8 7.8 0 0 1-1.7.7v.3a2 2 0 1 1-4 0v-.3a7.8 7.8 0 0 1-1.7-.7l-.2.2a2 2 0 1 1-2.8-2.8l.2-.2a7.8 7.8 0 0 1-.7-1.7H4a2 2 0 1 1 0-4h.3A7.8 7.8 0 0 1 5 7.5l-.2-.2a2 2 0 1 1 2.8-2.8l.2.2a7.8 7.8 0 0 1 1.7-.7V4a2 2 0 0 1 2-2Z"
                        fill="currentColor"
                        opacity=".3"
                      />
                      <circle cx="12" cy="12" r="3" fill="currentColor" />
                    </svg>
                  </span>
                </span>
                <span className="menu-title">Administrator</span>
                <span className="menu-arrow" />
              </button>

              <div
                className={`menu-sub menu-sub-accordion ${administratorOpen ? "show" : ""}`}
                style={{ display: administratorOpen ? "flex" : "none" }}
              >
                {visibleAdministratorMenus.map((menu) => <div className="menu-item" key={menu.code}>
                  <Link href={menu.href} className={`menu-link ${pathname === menu.href || pathname.startsWith(`${menu.href}/`) ? "active" : ""}`} onClick={onMobileClose}>
                    <span className="menu-bullet"><span className="bullet bullet-dot" /></span>
                    <span className="menu-title">{menu.title}</span>
                  </Link>
                </div>)}
              </div>
            </div>}
            {visibleCompanyMenus.length > 0 && <div
              className="menu-item menu-accordion"
            >
              <button
                type="button"
                className="menu-link border-0 w-100 bg-transparent text-start"
                aria-expanded={companiesOpen}
                onClick={() => setCompaniesOpen((open) => !open)}
              >
                <span className="menu-icon">
                  <span className="svg-icon svg-icon-2">
                    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path
                        d="M3 5a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5Z"
                        fill="currentColor"
                      />
                      <path
                        d="M3 8h18v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8Z"
                        fill="currentColor"
                        opacity=".3"
                      />
                    </svg>
                  </span>
                </span>
                <span className="menu-title">Companies</span>
                <span className="menu-arrow" />
              </button>

              <div
                className={`menu-sub menu-sub-accordion ${companiesOpen ? "show" : ""}`}
                style={{ display: companiesOpen ? "flex" : "none" }}
              >
                {visibleCompanyMenus.map((menu) => {
                  const active =
                    pathname === menu.href || pathname.startsWith(`${menu.href}/`);

                  return (
                    <div className="menu-item" key={menu.href}>
                      <Link
                        href={menu.href}
                        className={`menu-link ${active ? "active" : ""}`}
                        aria-current={active ? "page" : undefined}
                        onClick={onMobileClose}
                      >
                        <span className="menu-bullet">
                          <span className="bullet bullet-dot" />
                        </span>
                        <span className="menu-title">{menu.title}</span>
                      </Link>
                    </div>
                  );
                })}
              </div>
            </div>}

            {(visibleLandModules.length > 0 || canViewAcquisitionRequest || canViewAcquisitionProcesses) && <div className="menu-item pt-2">
              <div className="menu-content">
                <span className="menu-heading fw-bold text-uppercase fs-7">
                  Transaksi
                </span>
              </div>
            </div>}

            {visibleLandModules.length > 0 && <div className="menu-item menu-accordion">
              <button
                type="button"
                className="menu-link border-0 w-100 bg-transparent text-start"
                aria-expanded={landsOpen}
                onClick={() => setLandsOpen((open) => !open)}
              >
                <span className="menu-icon">
                  <span className="svg-icon svg-icon-2">
                    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M3 5a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5Z" fill="currentColor" />
                      <path d="M3 8h18v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8Z" fill="currentColor" opacity=".3" />
                    </svg>
                  </span>
                </span>
                <span className="menu-title">Lands</span>
                <span className="menu-arrow" />
              </button>
              <div
                className={`menu-sub menu-sub-accordion ${landsOpen ? "show" : ""}`}
                style={{ display: landsOpen ? "flex" : "none" }}
              >
                {visibleLandModules.map((item) => {
                  const href = moduleRoute(item);
                  const active = pathname === href || pathname.startsWith(`${href}/`);
                  return (
                    <div className="menu-item" key={item.id ?? item.module_code}>
                      <Link href={href} className={`menu-link ${active ? "active" : ""}`} aria-current={active ? "page" : undefined} onClick={onMobileClose}>
                        <span className="menu-bullet"><span className="bullet bullet-dot" /></span>
                        <span className="menu-title">{item.module_name}</span>
                      </Link>
                    </div>
                  );
                })}
              </div>
            </div>}

            {(canViewAcquisitionRequest || canViewAcquisitionProcesses) && <div className="menu-item menu-accordion">
              <button
                type="button"
                className="menu-link border-0 w-100 bg-transparent text-start"
                aria-expanded={acquisitionOpen}
                onClick={() => setAcquisitionOpen((open) => !open)}
              >
                <span className="menu-icon">
                  <span className="svg-icon svg-icon-2">
                    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
                      <path d="M7 3h8l4 4v13a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V5a2 2 0 0 1 2-2Z" fill="currentColor" opacity=".3" />
                      <path d="M14 3v5h5M8 12h8M8 16h6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  </span>
                </span>
                <span className="menu-title">Acquisition Process</span>
                <span className="menu-arrow" />
              </button>
              <div
                className={`menu-sub menu-sub-accordion ${acquisitionOpen ? "show" : ""}`}
                style={{ display: acquisitionOpen ? "flex" : "none" }}
              >
                {canViewAcquisitionRequest && <div className="menu-item">
                  <Link href="/admin/acquisition/request" prefetch={false} className={`menu-link ${pathname === "/admin/acquisition/request" ? "active" : ""}`} aria-current={pathname === "/admin/acquisition/request" ? "page" : undefined} onClick={onMobileClose}>
                    <span className="menu-bullet"><span className="bullet bullet-dot" /></span>
                    <span className="menu-title">Acquisition Request</span>
                  </Link>
                </div>}
                {canViewAcquisitionProcesses && acquisitionSequences.map((item) => {
                  const href = `/admin/acquisition/${acquisitionSlug(item.name) || `process-${item.id}`}`;
                  const active = pathname === href || pathname.startsWith(`${href}/`);
                  return (
                    <div className="menu-item" key={item.id}>
                      <Link href={href} prefetch={false} className={`menu-link ${active ? "active" : ""}`} aria-current={active ? "page" : undefined} onClick={onMobileClose}>
                        <span className="menu-bullet"><span className="bullet bullet-dot" /></span>
                        <span className="menu-title">{item.name.trim()}</span>
                      </Link>
                    </div>
                  );
                })}
                {canViewAcquisitionProcesses && acquisitionSequences.length === 0 && <div className="menu-item"><span className="menu-link text-muted"><span className="menu-title">{acquisitionLoading ? "Memuat proses..." : "Belum ada proses akuisisi."}</span></span></div>}
              </div>
            </div>}
            
            {visibleReportModules.length > 0 && <>
              <div className="menu-item pt-2">
                <div className="menu-content">
                  <span className="menu-heading fw-bold text-uppercase fs-7">Laporan</span>
                </div>
              </div>
              <div className="menu-item menu-accordion">
                <button type="button" className="menu-link border-0 w-100 bg-transparent text-start" aria-expanded={reportOpen} aria-controls="sidebar-report-links" onClick={() => setReportOpen((open) => !open)}>
                  <span className="menu-icon"><span className="svg-icon svg-icon-2"><svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M4 19V5M4 19h16M8 15l4-4 3 2 5-6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" /><circle cx="8" cy="15" r="1.5" fill="currentColor" /><circle cx="12" cy="11" r="1.5" fill="currentColor" /><circle cx="15" cy="13" r="1.5" fill="currentColor" /><circle cx="20" cy="7" r="1.5" fill="currentColor" /></svg></span></span>
                  <span className="menu-title">Report</span>
                  <span className="menu-arrow" />
                </button>
                <div id="sidebar-report-links" className={`menu-sub menu-sub-accordion ${reportOpen ? "show" : ""}`} style={{ display: reportOpen ? "flex" : "none" }}>
                  {visibleReportModules.map((item) => {
                    const href = reportPath(item.module_code) ?? "/admin/report";
                    const active = pathname === href || pathname.startsWith(`${href}/`);
                    return <div className="menu-item" key={item.id ?? item.module_code}>
                      <Link href={href} className={`menu-link ${active ? "active" : ""}`} aria-current={active ? "page" : undefined} onClick={onMobileClose}>
                        <span className="menu-bullet"><span className="bullet bullet-dot" /></span>
                        <span className="menu-title">{item.module_name}</span>
                      </Link>
                    </div>;
                  })}
                </div>
              </div>
            </>}

          </nav>
        </div>
      </div>

      <div
        className="aside-footer flex-column-auto px-5 py-5"
        id="kt_aside_footer">
      </div>
    </aside>
  );
}
