import Link from "next/link";
import Toolbar from "./toolbar";

type HeaderProps = {
  mobileSidebarOpen: boolean;
  desktopSidebarMinimized: boolean;
  onToggleDesktopSidebar: () => void;
  onToggleMobileSidebar: () => void;
  onLogout: () => void;
};

export default function Header({
  mobileSidebarOpen,
  desktopSidebarMinimized,
  onToggleDesktopSidebar,
  onToggleMobileSidebar,
  onLogout,
}: HeaderProps) {
  return (
    <header
      id="kt_header"
      className="header align-items-stretch"
    >
      <div className="header-brand">
        <div
          className="d-flex align-items-center d-lg-none ms-n3 me-1"
          title={mobileSidebarOpen ? "Tutup menu" : "Tampilkan menu"}
        >
          <button
            type="button"
            className="btn btn-icon btn-active-color-primary w-30px h-30px"
            id="kt_aside_mobile_toggle"
            aria-label={mobileSidebarOpen ? "Tutup menu navigasi" : "Buka menu navigasi"}
            aria-expanded={mobileSidebarOpen}
            onClick={onToggleMobileSidebar}
          >
            <span className="svg-icon svg-icon-1">
              <svg
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                aria-hidden="true"
              >
                <path
                  d="M21 7H3C2.4 7 2 6.6 2 6V4C2 3.4 2.4 3 3 3H21C21.6 3 22 3.4 22 4V6C22 6.6 21.6 7 21 7Z"
                  fill="currentColor"
                />
                <path
                  opacity=".3"
                  d="M21 14H3C2.4 14 2 13.6 2 13V11C2 10.4 2.4 10 3 10H21C21.6 10 22 10.4 22 11V13C22 13.6 21.6 14 21 14ZM22 20V18C22 17.4 21.6 17 21 17H3C2.4 17 2 17.4 2 18V20C2 20.6 2.4 21 3 21H21C21.6 21 22 20.6 22 20Z"
                  fill="currentColor"
                />
              </svg>
            </span>
          </button>
        </div>
        <Link
          href="/admin/dashboard"
          className="header-brand-link d-flex align-items-center text-white text-decoration-none"
          aria-label="AEP Plantations"
        >
          <img src="/metronic/upload/aeplogonew.png" alt="" className="header-brand-logo" />
          <span className="text-white fw-bold fs-18 ms-2">GIS - AEP Plantations</span>
        </Link>
        <button
          type="button"
          className={`btn btn-icon w-auto px-0 btn-active-color-primary aside-minimize d-none d-lg-flex ${desktopSidebarMinimized ? "active" : ""}`}
          id="kt_aside_toggle"
          aria-label={desktopSidebarMinimized ? "Tampilkan sidebar" : "Sembunyikan sidebar"}
          aria-pressed={desktopSidebarMinimized}
          onClick={onToggleDesktopSidebar}
        >
          <span className="svg-icon svg-icon-1 me-n1 minimize-default">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <rect opacity=".3" x="8.5" y="11" width="12" height="2" rx="1" fill="currentColor" />
              <path d="M10.37 11.69 12.12 10.23a1 1 0 0 0-1.36-1.47L7.84 11.25a1 1 0 0 0 0 1.5l2.92 2.6a1 1 0 1 0 1.36-1.47l-1.75-1.46a.5.5 0 0 1 0-.73Z" fill="currentColor" />
              <path opacity=".5" d="M16 5v1a1 1 0 0 1-1 1H6v10h9a1 1 0 0 1 1 1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2Z" fill="currentColor" />
            </svg>
          </span>
          <span className="svg-icon svg-icon-1 minimize-active">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <rect opacity=".3" width="12" height="2" rx="1" transform="matrix(-1 0 0 1 15.5 11)" fill="currentColor" />
              <path d="m13.63 11.69-1.75-1.46a1 1 0 0 1 1.36-1.47l2.92 2.49a1 1 0 0 1 0 1.5l-2.92 2.6a1 1 0 1 1-1.36-1.47l1.75-1.46a.5.5 0 0 0 0-.73Z" fill="currentColor" />
              <path d="M8 5v1a1 1 0 0 0 1 1h9v10H9a1 1 0 0 0-1 1v1a2 2 0 0 0 2 2h9a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2h-9a2 2 0 0 0-2 2Z" fill="currentColor" />
            </svg>
          </span>
        </button>
      </div>

      <Toolbar onLogout={onLogout} />
    </header>
  );
}
