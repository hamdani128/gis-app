"use client";

import { useState } from "react";
import type { FormEvent } from "react";
import Link from "next/link";
import { useUpdateUserProfile, useUserProfile } from "./user-profile-context";
import type { UserProfile } from "./user-profile-context";
import AccountIcon from "./account-icon";
import styles from "./account.module.css";

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

export default function AccountSettings() {
  const profile = useUserProfile();
  const updateProfile = useUpdateUserProfile();
  const [name, setName] = useState(profile.name);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showPasswords, setShowPasswords] = useState(false);
  const changePassword = Boolean(currentPassword || newPassword || confirmation);
  const initials = profile.name.trim().split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase() || "U";
  const hasChanges = name.trim() !== profile.name || changePassword;

  function resetForm() {
    setName(profile.name);
    setCurrentPassword("");
    setNewPassword("");
    setConfirmation("");
    setError("");
    setSuccess("");
    setShowPasswords(false);
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (saving) return;
    setError("");
    setSuccess("");
    if (!name.trim()) { setError("Nama wajib diisi."); return; }
    if (changePassword && (!currentPassword || newPassword.length < 8)) {
      setError("Isi password saat ini dan password baru minimal 8 karakter.");
      return;
    }
    if (changePassword && newPassword !== confirmation) {
      setError("Konfirmasi password baru tidak cocok.");
      return;
    }
    setSaving(true);
    try {
      const response = await fetch(`${apiBase}/api/auth/me`, {
        method: "PUT",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim(), ...(changePassword ? { current_password: currentPassword, new_password: newPassword } : {}) }),
      });
      const result = await response.json() as { message?: string; data?: UserProfile };
      if (!response.ok || !result.data) throw new Error(response.status === 401 ? "Sesi berakhir. Silakan login kembali." : result.message || "Pengaturan akun gagal disimpan.");
      updateProfile(result.data);
      setName(result.data.name);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmation("");
      setSuccess(changePassword ? "Nama dan password berhasil diperbarui." : "Nama berhasil diperbarui.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Tidak dapat terhubung ke server.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className={styles.page}>
      <nav className={styles.breadcrumb} aria-label="Breadcrumb">
        <Link href="/admin/dashboard">Dashboard</Link><span aria-hidden="true">/</span><span>Akun saya</span>
      </nav>
      <div className={styles.pageHeading}>
        <div><span className={styles.eyebrow}>AKUN SAYA</span><h1>Pengaturan akun</h1><p>Kelola informasi profil dan keamanan akun Anda.</p></div>
        <span className={styles.status}><span />Sesi aktif</span>
      </div>
      <div className={styles.accountGrid}>
        <aside className={styles.profileCard} aria-label="Ringkasan profil">
          <div className={styles.profileCover}><span>GIS · AEP PLANTATIONS</span><AccountIcon name="leaf" /></div>
          <div className={styles.profileBody}>
            <div className={styles.largeAvatar}>{initials}<span><AccountIcon name="check" /></span></div>
            <h2>{profile.name}</h2><p className={styles.handle}>@{profile.username}</p>
            <span className={styles.departmentBadge}>{profile.department || "Departemen belum diatur"}</span>
            <div className={styles.profileDetails}>
              <div><AccountIcon name="mail" /><span><small>Email</small><strong>{profile.email || "Belum diatur"}</strong></span></div>
              <div><AccountIcon name="building" /><span><small>Departemen</small><strong>{profile.department || "Belum diatur"}</strong></span></div>
              {profile.process_role && <div><AccountIcon name="user" /><span><small>Peran proses</small><strong>{profile.process_role}</strong></span></div>}
            </div>
            <div className={styles.profileNote}><AccountIcon name="shield" /><span>Akun terhubung dengan akses GIS AEP Plantations.</span></div>
          </div>
        </aside>
        <form className={styles.form} onSubmit={save}>
          {error && <div className="alert alert-danger mb-0" role="alert">{error}</div>}
          {success && <div className="alert alert-success mb-0" role="status">{success}</div>}
          <section className={styles.section} aria-labelledby="account-profile-title">
            <div className={styles.sectionHeading}><span className={styles.sectionIcon}><AccountIcon name="user" /></span><div><h2 id="account-profile-title">Informasi profil</h2><p>Nama yang tampil di aplikasi dan panel pengguna.</p></div></div>
            <fieldset disabled={saving} className={styles.fields}>
              <div className={styles.field}>
                <label htmlFor="account-name">Nama tampilan <span>*</span></label>
                <input id="account-name" value={name} onChange={(event) => setName(event.target.value)} required maxLength={100} autoComplete="name" />
              </div>
              <div className={styles.twoColumns}>
                <div className={styles.field}><label htmlFor="account-username">Username</label><input id="account-username" value={profile.username} readOnly autoComplete="username" /><small>Identitas login akun Anda.</small></div>
                <div className={styles.field}><label htmlFor="account-email">Email</label><input id="account-email" value={profile.email || "Belum diatur"} readOnly /><small>Dikelola oleh administrator.</small></div>
              </div>
            </fieldset>
          </section>
          <section className={styles.section} aria-labelledby="account-security-title">
            <div className={styles.sectionHeading}><span className={styles.sectionIcon}><AccountIcon name="lock" /></span><div><h2 id="account-security-title">Keamanan akun</h2><p>Perbarui password untuk menjaga akses akun Anda.</p></div></div>
            <fieldset disabled={saving} className={styles.fields}>
              <div className={styles.passwordHint}><AccountIcon name="shield" /><p id="account-password-help">Gunakan minimal 8 karakter. Kosongkan bagian ini jika hanya ingin mengganti nama.</p></div>
              <div className={styles.field}>
                <label htmlFor="account-current-password">Password saat ini</label>
                <input id="account-current-password" type={showPasswords ? "text" : "password"} value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} autoComplete="current-password" maxLength={1024} required={changePassword} placeholder="Masukkan password saat ini" />
              </div>
              <div className={styles.twoColumns}>
                <div className={styles.field}><label htmlFor="account-new-password">Password baru</label><input id="account-new-password" type={showPasswords ? "text" : "password"} value={newPassword} onChange={(event) => setNewPassword(event.target.value)} autoComplete="new-password" minLength={8} maxLength={100} required={changePassword} aria-describedby="account-password-help" placeholder="Minimal 8 karakter" /></div>
                <div className={styles.field}><label htmlFor="account-password-confirmation">Konfirmasi password baru</label><input id="account-password-confirmation" type={showPasswords ? "text" : "password"} value={confirmation} onChange={(event) => setConfirmation(event.target.value)} autoComplete="new-password" maxLength={100} required={changePassword} placeholder="Ulangi password baru" /></div>
              </div>
              <label className={styles.showPassword}><input type="checkbox" checked={showPasswords} onChange={(event) => setShowPasswords(event.target.checked)} />Tampilkan password</label>
            </fieldset>
          </section>
          <div className={styles.formFooter}>
            <span><AccountIcon name="shield" />Perubahan hanya berlaku untuk akun Anda.</span>
            <div><button type="button" className={styles.resetButton} disabled={saving || !hasChanges} onClick={resetForm}>Batalkan</button><button type="submit" className={styles.saveButton} disabled={saving || !hasChanges}><AccountIcon name="check" />{saving ? "Menyimpan..." : "Simpan perubahan"}</button></div>
          </div>
        </form>
      </div>
    </div>
  );
}
