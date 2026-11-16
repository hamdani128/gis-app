"use client";

import { useRef, useState } from "react";
import type { KeyboardEvent } from "react";
import AccountIcon from "../auth/account-icon";
import styles from "./company-picker.module.css";

type Company = { id: number; name: string; is_primary: boolean };
type CompanyPickerProps = {
  companies: Company[];
  selectedID: string;
  open: boolean;
  onToggle: () => void;
  onSelect: (id: number) => void;
};

function companyInitials(name: string) {
  return name.replace(/^PT\.?\s+/i, "").trim().split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase() || "CO";
}

export default function CompanyPicker({ companies, selectedID, open, onToggle, onSelect }: CompanyPickerProps) {
  const [search, setSearch] = useState("");
  const triggerRef = useRef<HTMLButtonElement>(null);
  const optionsRef = useRef<HTMLDivElement>(null);
  const selected = companies.find((company) => String(company.id) === selectedID);
  const filtered = companies.filter((company) => company.name.toLocaleLowerCase("id").includes(search.trim().toLocaleLowerCase("id")));

  function navigateOptions(event: KeyboardEvent<HTMLElement>) {
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
    const buttons = Array.from(optionsRef.current?.querySelectorAll<HTMLButtonElement>("button") ?? []);
    if (buttons.length === 0) return;
    event.preventDefault();
    const index = buttons.indexOf(event.target as HTMLButtonElement);
    const next = event.key === "Home" ? 0 : event.key === "End" ? buttons.length - 1
      : event.key === "ArrowDown" ? (index + 1) % buttons.length
      : index <= 0 ? buttons.length - 1 : index - 1;
    buttons[next].focus();
  }

  return <>
    <button
      ref={triggerRef}
      type="button"
      className={styles.trigger}
      aria-label={`Company aktif: ${selected?.name || "Pilih company"}`}
      aria-haspopup="dialog"
      aria-expanded={open}
      aria-controls="kt_header_company_menu"
      onClick={() => { setSearch(""); onToggle(); }}
    >
      <span className={styles.triggerIcon}><AccountIcon name="building" /></span>
      <span className={styles.triggerCopy}>
        <span className={styles.triggerLabel}><span />Company aktif</span>
        <strong title={selected?.name}>{selected?.name || "Pilih company"}</strong>
      </span>
      <span className={styles.chevron}><svg width="14" height="14" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m6 9 6 6 6-6" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg></span>
    </button>
    {open && <div
      id="kt_header_company_menu"
      className={styles.panel}
      role="dialog"
      aria-label="Pilih company aktif"
      onKeyDown={(event) => {
        if (event.key === "Escape") { event.preventDefault(); triggerRef.current?.focus(); }
      }}
    >
      <div className={styles.panelHeader}>
        <div><span className={styles.eyebrow}>WORKSPACE</span><h2>Pilih company</h2><p>Beralih ke company yang ingin Anda kelola.</p></div>
        <span className={styles.companyCount}>{companies.length}<small>company</small></span>
      </div>
      <div className={styles.searchWrap}>
        <svg width="17" height="17" viewBox="0 0 24 24" fill="none" aria-hidden="true"><circle cx="10.5" cy="10.5" r="6.5" stroke="currentColor" strokeWidth="1.8" /><path d="m16 16 4 4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" /></svg>
        <input type="search" aria-label="Cari company" placeholder="Cari nama company..." value={search} onChange={(event) => setSearch(event.target.value)} onKeyDown={(event) => { if (event.key === "ArrowDown") navigateOptions(event); }} autoFocus />
      </div>
      <div className={styles.listLabel}>COMPANY YANG DAPAT DIAKSES</div>
      <div ref={optionsRef} className={styles.options} role="group" aria-label="Daftar company" onKeyDown={navigateOptions}>
        {filtered.map((company) => {
          const active = String(company.id) === selectedID;
          return <button
            key={company.id}
            type="button"
            className={`${styles.option} ${active ? styles.selected : ""}`}
            aria-pressed={active}
            onClick={() => { onSelect(company.id); triggerRef.current?.focus(); }}
          >
            <span className={styles.monogram} aria-hidden="true">{companyInitials(company.name)}</span>
            <span className={styles.optionCopy}><strong>{company.name || `Company ${company.id}`}</strong><small>{active ? "Sedang digunakan" : "Beralih ke company ini"}</small></span>
            {company.is_primary && <span className={styles.primaryBadge}>Utama</span>}
            <span className={styles.selectionMark} aria-hidden="true">{active && <AccountIcon name="check" />}</span>
          </button>;
        })}
        {filtered.length === 0 && <div className={styles.empty} role="status"><AccountIcon name="building" /><strong>Company tidak ditemukan</strong><span>Coba gunakan kata kunci lain.</span></div>}
      </div>
      <div className={styles.footer}><AccountIcon name="shield" /><span>Akses company sesuai dengan akun Anda.</span></div>
    </div>}
  </>;
}
