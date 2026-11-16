"use client";

import { useState } from "react";
import type { FormEvent } from "react";
import Link from "next/link";
import AuthShell from "../auth-shell";
import styles from "../auth.module.css";
import AccountIcon from "../../../components/auth/account-icon";

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

export default function ForgotPasswordPage() {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting) return;
    setError("");
    setSubmitting(true);
    try {
      const response = await fetch(`${apiBase}/api/auth/forgot-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: username.trim(), email: email.trim() }),
      });
      const result = await response.json().catch(() => null);
      if (!response.ok) throw new Error(result?.message || "Permintaan reset belum dapat diproses. Silakan coba kembali.");
      setSent(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Tidak dapat terhubung ke server.");
    } finally {
      setSubmitting(false);
    }
  }

  return <AuthShell>
    <div className={styles.resetIcon}><AccountIcon name={sent ? "mail" : "lock"} /></div>
    <span className={styles.eyebrow}>ACCOUNT RECOVERY</span>
    <h1 className={styles.heading}>{sent ? "Periksa email Anda." : "Lupa password?"}</h1>
    <p className={styles.intro}>{sent ? "Jika username dan email cocok dengan akun aktif, password baru akan dikirim ke email yang terdaftar. Periksa juga folder spam." : "Masukkan username dan email yang terdaftar. Kami akan mengirimkan password baru ke email Anda."}</p>
    {error && <div className={styles.message} role="alert">{error}</div>}
    {sent ? <>
      <div className={`${styles.message} ${styles.success}`} role="status">Setelah berhasil masuk, segera ganti password melalui menu Pengaturan akun.</div>
      <Link className={styles.submit} href="/auth/login">Kembali ke login<AccountIcon name="arrow" /></Link>
    </> : <>
      <form className={styles.form} onSubmit={submit} aria-busy={submitting}>
        <div className={styles.field}><label htmlFor="reset-username">Username</label><div className={styles.inputWrap}><AccountIcon name="user" /><input id="reset-username" autoComplete="username" placeholder="Masukkan username Anda" value={username} onChange={(event) => setUsername(event.target.value)} required maxLength={254} disabled={submitting} autoFocus /></div></div>
        <div className={styles.field}><label htmlFor="reset-email">Email terdaftar</label><div className={styles.inputWrap}><AccountIcon name="mail" /><input id="reset-email" type="email" autoComplete="email" placeholder="nama@aepplantations.id" value={email} onChange={(event) => setEmail(event.target.value)} required maxLength={254} disabled={submitting} /></div></div>
        <button type="submit" className={styles.submit} disabled={submitting}>{submitting ? <><span className="spinner-border spinner-border-sm" aria-hidden="true" />Memproses permintaan...</> : <>Kirim password baru<AccountIcon name="arrow" /></>}</button>
      </form>
      <Link href="/auth/login" className={styles.backLink}><AccountIcon name="arrow" />Kembali ke login</Link>
    </>}
    <div className={styles.help}>Tidak lagi memiliki akses ke email terdaftar?<br /><strong>Hubungi Departemen ICT untuk pemulihan akun.</strong></div>
  </AuthShell>;
}
