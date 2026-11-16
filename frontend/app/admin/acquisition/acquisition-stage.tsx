"use client";

import type { ReactNode } from "react";
import styles from "./acquisition-workspace.module.css";

export function ProcessApprovalBadge({ approval }: { approval: number | null }) {
  const color = approval === 1 ? "success" : approval === 0 ? "danger" : approval === 2 ? "warning" : "secondary";
  const title = approval === 1 ? "Approved" : approval === 0 ? "Rejected" : approval === 2 ? "On Hold" : "Belum ada keputusan";
  return <span className={`badge badge-light-${color}`}>{title}</span>;
}

export default function AcquisitionStage({ title, number, count, active, children }: { title: string; number: number; count: number; active: boolean; children: ReactNode }) {
  return <details className={`submit-memo-step ${styles.stage}${count ? " has-record" : ""}${active ? ` ${styles.activeStage}` : ""}`} data-step-number={number} open={active || undefined}>
    <summary className={styles.stageHeading}><span><strong>{title}</strong><small>{active ? "Tahap aktif · siap diproses" : count ? `${count} catatan proses` : "Belum ada catatan"}</small></span><span className={active ? styles.activeBadge : styles.countBadge}>{active ? "Aktif" : count}</span></summary>
    <div className={styles.stageBody}>{children}</div>
  </details>;
}
