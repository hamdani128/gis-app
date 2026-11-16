"use client";

import { useState } from "react";
import type { FormEvent } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import AuthShell from "./auth-shell";
import AccountIcon from "../../components/auth/account-icon";
import styles from "./auth.module.css";

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const sessionActivityKey = "gis.auth.last-activity";
const sessionLogoutKey = "gis.auth.logged-out";

export default function LoginPage({ idleExpired = false }: { idleExpired?: boolean }) {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const notice = idleExpired ? "Sesi berakhir karena tidak ada aktivitas selama 1 jam. Silakan masuk kembali." : "";
  const [submitting, setSubmitting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;
    setError("");
    setSubmitting(true);

    try {
      const response = await fetch(`${apiBase}/api/auth/login`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ identifier: email.trim(), password }),
      });
      const result = await response.json().catch(() => null);

      if (!response.ok) {
        setError(result?.message ?? "Email atau kata sandi tidak valid.");
        return;
      }

      localStorage.setItem(sessionActivityKey, String(Date.now()));
      localStorage.removeItem(sessionLogoutKey);
      router.replace("/admin/dashboard");
    } catch {
      setError("Tidak dapat terhubung ke server. Pastikan backend sedang berjalan.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthShell>
      <span className={styles.eyebrow}>WELCOME TO AEP GIS</span>
      <h1 className={styles.heading}>Selamat datang kembali.</h1>
      <p className={styles.intro}>Masuk ke akun Anda untuk mengakses informasi lahan dan mengelola aktivitas perkebunan.</p>
      {error && <div className={styles.message} role="alert">{error}</div>}
      {notice && <div className={`${styles.message} ${styles.notice}`} role="status">{notice}</div>}
      <form className={styles.form} onSubmit={handleSubmit} aria-busy={submitting}>
        <div className={styles.field}>
          <label htmlFor="login-email">Username atau email</label>
          <div className={styles.inputWrap}><AccountIcon name="user" /><input id="login-email" type="text" name="identifier" autoComplete="username" placeholder="Masukkan username atau email" value={email} onChange={(event) => setEmail(event.target.value)} required maxLength={254} disabled={submitting} autoFocus /></div>
        </div>
        <div className={styles.field}>
          <div className={styles.labelRow}><label htmlFor="login-password">Password</label><Link href="/auth/forgot-password">Lupa password?</Link></div>
          <div className={styles.inputWrap}><AccountIcon name="lock" /><input id="login-password" type={showPassword ? "text" : "password"} name="password" autoComplete="current-password" placeholder="Masukkan password Anda" value={password} onChange={(event) => setPassword(event.target.value)} required maxLength={1024} disabled={submitting} /><button className={styles.eyeButton} type="button" aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"} aria-pressed={showPassword} onClick={() => setShowPassword((visible) => !visible)}><svg width="17" height="17" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z" stroke="currentColor" strokeWidth="1.6" /><circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth="1.6" />{showPassword && <path d="m3 3 18 18" stroke="currentColor" strokeWidth="1.6" />}</svg></button></div>
        </div>
        <button className={styles.submit} type="submit" disabled={submitting}>{submitting ? <><span className="spinner-border spinner-border-sm" aria-hidden="true" />Memeriksa akun...</> : <>Masuk ke dashboard<AccountIcon name="arrow" /></>}</button>
      </form>
      <p className={styles.accessNote}><AccountIcon name="shield" />Akses hanya untuk pengguna yang terdaftar.</p>
      <div className={styles.help}>Mengalami kendala saat masuk?<br /><strong>Hubungi Departemen ICT untuk bantuan.</strong></div>
    </AuthShell>
  );
}
