import type { ReactNode } from "react";
import Image from "next/image";
import AccountIcon from "../../components/auth/account-icon";
import styles from "./auth.module.css";

export default function AuthShell({ children }: { children: ReactNode }) {
  return <main className={styles.shell}>
    <aside className={styles.visual} aria-label="GIS AEP Plantations">
      <div className={styles.brand}><Image src="/metronic/upload/aeplogonew.png" width={49} height={55} alt="AEP Plantations" /><span><strong>AEP Plantations</strong><small>GEOGRAPHIC INFORMATION SYSTEM</small></span></div>
      <div className={styles.visualContent}>
        <span className={styles.visualEyebrow}><span />LAND & PLANTATION MANAGEMENT</span>
        <h2>One landscape.<br />A clearer perspective.</h2>
        <p>Informasi lahan, pemetaan, dan proses akuisisi dalam satu ruang kerja yang terhubung.</p>
        <div className={styles.featureRow}><span><AccountIcon name="leaf" />Pemetaan lahan</span><span><AccountIcon name="building" />Multi company</span><span><AccountIcon name="shield" />Akses terkelola</span></div>
      </div>
      <div className={styles.visualFooter}><span>GIS · AEP PLANTATIONS</span><span>Connected land. Informed decisions.</span></div>
    </aside>
    <section className={styles.formSide}>
      <div className={styles.mobileBrand}><Image src="/metronic/upload/aeplogonew.png" width={28} height={33} alt="" /><strong>AEP Plantations</strong></div>
      <div className={styles.formWrap}>{children}</div>
      <footer className={styles.footer}><span>&copy; {new Date().getFullYear()} AEP Plantations</span><span>GIS Management System</span></footer>
    </section>
  </main>;
}
