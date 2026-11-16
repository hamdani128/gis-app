"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useUserProfile } from "../auth/user-profile-context";
import AccountIcon from "../auth/account-icon";
import accountStyles from "../auth/account.module.css";
import CompanyPicker from "./company-picker";

type SweetAlertResult = { isConfirmed: boolean };
type SweetAlertOptions = {
  title?: string;
  text?: string;
  icon?: "warning" | "success" | "error";
  showCancelButton?: boolean;
  confirmButtonText?: string;
  cancelButtonText?: string;
  reverseButtons?: boolean;
  buttonsStyling?: boolean;
  customClass?: {
    popup?: string;
    title?: string;
    htmlContainer?: string;
    confirmButton?: string;
    cancelButton?: string;
  };
};

declare global {
  interface Window {
    Swal?: {
      fire: (options: SweetAlertOptions) => Promise<SweetAlertResult>;
    };
  }
}

type ToolbarProps = {
  onLogout: () => void;
};

type AccessibleCompany = {
  id: number;
  name: string;
  is_primary: boolean;
};

type APIEnvelope<T> = { message?: string; data?: T };

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

export default function Toolbar({ onLogout }: ToolbarProps) {
  const pathname = usePathname();
  const profile = useUserProfile();
  const menuRef = useRef<HTMLDivElement>(null);
  const companyMenuRef = useRef<HTMLDivElement>(null);
  const [accountMenuOpen, setAccountMenuOpen] = useState(false);
  const [companyMenuOpen, setCompanyMenuOpen] = useState(false);
  const [accessibleCompanies, setAccessibleCompanies] = useState<AccessibleCompany[]>([]);
  const [selectedCompanyID, setSelectedCompanyID] = useState("");

  useEffect(() => {
    let active = true;
    fetch(`${apiBase}/api/auth/companies`, { credentials: "include", cache: "no-store" })
      .then(async (response) => {
        if (!response.ok) throw new Error("Company list is unavailable");
        const result = (await response.json()) as APIEnvelope<AccessibleCompany[]>;
        if (active) {
          const companies = Array.isArray(result?.data) ? result.data : [];
          setAccessibleCompanies(companies);
          const savedID = window.localStorage.getItem(`gis.selected-company.${profile.username.toLowerCase()}`);
          const selected = companies.find((company) => String(company.id) === savedID)
            ?? companies.find((company) => company.is_primary)
            ?? companies[0];
          setSelectedCompanyID(selected ? String(selected.id) : "");
        }
      })
      .catch(() => { if (active) setAccessibleCompanies([]); });
    return () => { active = false; };
  }, [profile.username]);

  const isDashboard = pathname === "/admin/dashboard" || pathname === "/admin/dashboard/";
  const title = pathname.startsWith("/admin/users")
    ? "Pengguna"
    : pathname.startsWith("/admin/companies/info")
      ? "Companies Info"
    : pathname.startsWith("/admin/companies/region")
      ? "Region"
    : pathname.startsWith("/admin/companies/land-info")
      ? "Land Permits"
    : pathname.startsWith("/admin/lands/hvclands")
      ? "HCV Lands"
    : pathname.startsWith("/admin/lands/potentiallands")
      ? "Potential Lands"
    : pathname.startsWith("/admin/lands/divlands")
      ? "Block Plans"
    : pathname.startsWith("/admin/lands/landowner")
      ? "Land Owners"
    : pathname.startsWith("/admin/lands/canceledland")
      ? "Canceled Lands"
    : pathname.startsWith("/admin/lands")
      ? "Lands"
    : pathname.startsWith("/admin/acquisition/request")
      ? "Acquisition Request"
    : pathname.startsWith("/admin/acquisition")
      ? "Acquisition Process"
    : pathname.startsWith("/admin/administrator/ref-value-setting")
      ? "Ref Value Setting"
      : pathname.startsWith("/admin/administrator/module")
        ? "Module Setting"
      : pathname.startsWith("/admin/administrator/doc-type")
        ? "Doc Type Setting"
      : pathname.startsWith("/admin/administrator/officer")
        ? "Officer"
      : pathname.startsWith("/admin/administrator/bussiness-process-officer")
        ? "Business Process Officer"
      : pathname.startsWith("/admin/administrator/bussiness-process")
        ? "Business Process"
      : "Dashboard";
  const showPageTitle = !isDashboard
    && !pathname.startsWith("/admin/account")
    && !pathname.startsWith("/admin/report")
    && !pathname.startsWith("/admin/administrator/ref-value-setting")
    && !pathname.startsWith("/admin/companies/info")
    && !pathname.startsWith("/admin/companies/region")
    && !pathname.startsWith("/admin/companies/land-info")
    && !pathname.startsWith("/admin/lands")
    && !pathname.startsWith("/admin/acquisition")
    && !pathname.startsWith("/admin/administrator/module")
    && !pathname.startsWith("/admin/administrator/doc-type")
    && !pathname.startsWith("/admin/administrator/officer")
    && !pathname.startsWith("/admin/administrator/bussiness-process");
  const initials = profile.name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0])
    .join("")
    .toUpperCase() || "U";
  const selectedCompany = accessibleCompanies.find((company) => String(company.id) === selectedCompanyID) ?? null;
  useEffect(() => {
    if (!selectedCompany) return;
    window.localStorage.setItem("gis.selected-company." + profile.username.toLowerCase(), String(selectedCompany.id));
    window.dispatchEvent(new CustomEvent("gis:company-change", { detail: { id: selectedCompany.id, name: selectedCompany.name } }));
  }, [selectedCompany, profile.username]);
  const companySelector = accessibleCompanies.length > 0 ? (
    <div className="position-relative flex-shrink-0" ref={companyMenuRef}>
      <CompanyPicker
        companies={accessibleCompanies}
        selectedID={selectedCompanyID}
        open={companyMenuOpen}
        onToggle={() => { setCompanyMenuOpen((open) => !open); setAccountMenuOpen(false); }}
        onSelect={(id) => {
          setSelectedCompanyID(String(id));
          setCompanyMenuOpen(false);
        }}
      />
    </div>
  ) : null;

  useEffect(() => {
    if (!accountMenuOpen && !companyMenuOpen) return;

    function closeOnOutsideClick(event: PointerEvent) {
      if (event.target instanceof Node
        && !menuRef.current?.contains(event.target)
        && !companyMenuRef.current?.contains(event.target)) {
        setAccountMenuOpen(false);
        setCompanyMenuOpen(false);
      }
    }

    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setAccountMenuOpen(false);
        setCompanyMenuOpen(false);
      }
    }

    document.addEventListener("pointerdown", closeOnOutsideClick);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("pointerdown", closeOnOutsideClick);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [accountMenuOpen, companyMenuOpen]);

  async function handleLogout() {
    setAccountMenuOpen(false);
    setCompanyMenuOpen(false);
    const result = window.Swal
      ? await window.Swal.fire({
          title: "Keluar dari aplikasi?",
          text: "Sesi Anda akan diakhiri dan Anda akan kembali ke halaman login.",
          icon: "warning",
          showCancelButton: true,
          confirmButtonText: "Ya, keluar",
          cancelButtonText: "Batal",
          reverseButtons: true,
          buttonsStyling: false,
          customClass: {
            popup: "rounded-4 shadow-lg",
            title: "fw-bold text-gray-900",
            htmlContainer: "text-muted",
            confirmButton: "btn btn-dark ms-2",
            cancelButton: "btn btn-light me-2",
          },
        })
      : { isConfirmed: window.confirm("Keluar dari aplikasi dan akhiri sesi Anda?") };

    if (result.isConfirmed) onLogout();
  }

  return (
    <div className="toolbar d-flex align-items-stretch">
      <div className="container-fluid py-4 py-lg-0 d-flex align-items-center justify-content-between gap-4">
        {companySelector}
        {showPageTitle && (
          <div className="page-title d-flex flex-column me-auto">
            <h1 className="text-gray-900 fw-bold fs-3 mb-0">{title}</h1>
            <ul className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 pt-1 mb-0">
              <li className="breadcrumb-item">
                <Link href="/admin/dashboard" className="text-muted text-hover-primary">
                  Admin
                </Link>
              </li>
              <li className="breadcrumb-item">
                <span className="bullet bg-gray-300 w-5px h-2px" />
              </li>
              <li className="breadcrumb-item text-muted">{title}</li>
            </ul>
          </div>
        )}

        <div className="d-flex align-items-center gap-2 ms-auto">
        <div className="position-relative flex-shrink-0" ref={menuRef}>
          <button
            type="button"
            className={accountStyles.trigger}
            aria-label="Buka menu akun"
            aria-haspopup="menu"
            aria-expanded={accountMenuOpen}
            aria-controls="kt_header_account_menu"
            onClick={() => { setAccountMenuOpen((open) => !open); setCompanyMenuOpen(false); }}
          >
            <span className={accountStyles.triggerAvatar}>{initials}<span /></span>
            <span className="d-none d-sm-flex flex-column ms-3 me-2 toolbar-profile-info">
              <span className="text-gray-900 fw-bold fs-7 text-truncate" title={profile.name}>{profile.name}</span>
              <span className="text-muted fw-semibold fs-8 text-truncate" title={profile.process_role || profile.department}>{profile.process_role || profile.department || "Next GIS"}</span>
            </span>
            <svg className="ms-2 text-muted" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="m6 9 6 6 6-6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </button>

          {accountMenuOpen && (
            <div
              id="kt_header_account_menu"
              className={accountStyles.menu}
              role="menu"
              aria-label="Menu akun"
            >
              <div className={accountStyles.menuHeader}>
                <div className={accountStyles.menuTopline}><span>AKUN SAYA</span><span className={accountStyles.status}><span />Sesi aktif</span></div>
                <div className={accountStyles.menuIdentity}>
                  <span className={accountStyles.menuAvatar}>{initials}</span>
                  <div><strong title={profile.name}>{profile.name}</strong><span title={profile.email || profile.username}>{profile.email || `@${profile.username}`}</span></div>
                </div>
                <div className={accountStyles.menuDepartment}><AccountIcon name="building" /><span>{profile.department || "Departemen belum diatur"}</span></div>
                {profile.process_role && <div className={accountStyles.menuRole}>{profile.process_role}</div>}
              </div>
              <div className={accountStyles.menuActions}>
              <Link
                href="/admin/account"
                className={accountStyles.menuAction}
                role="menuitem"
                onClick={() => setAccountMenuOpen(false)}
                aria-current={pathname === "/admin/account" ? "page" : undefined}
              >
                <span className={accountStyles.actionIcon}><AccountIcon name="user" /></span>
                <span className={accountStyles.actionCopy}><strong>Pengaturan akun</strong><small>Kelola profil dan password</small></span>
                <AccountIcon name="arrow" />
              </Link>
              <div className={accountStyles.menuDivider} />
              <button
                type="button"
                className={`${accountStyles.menuAction} ${accountStyles.logoutAction}`}
                role="menuitem"
                onClick={handleLogout}
              >
                <span className={accountStyles.actionIcon}><AccountIcon name="logout" /></span>
                <span className={accountStyles.actionCopy}><strong>Keluar dari aplikasi</strong><small>Akhiri sesi Anda</small></span>
              </button>
              </div>
              <div className={accountStyles.menuFooter}><AccountIcon name="leaf" />GIS · AEP Plantations</div>
            </div>
          )}
        </div>
        </div>
      </div>
    </div>
  );
}
