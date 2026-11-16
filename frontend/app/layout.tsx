import type { Metadata } from "next";
import Script from "next/script";
import type { ReactNode } from "react";
import "./globals.css";

export const metadata: Metadata = {
  title: "AEP Plantations - GIS APP",
  description: "Aplikasi GIS",
  icons: {
    icon: "/metronic/upload/aeplogonew.png",
    shortcut: "/metronic/upload/aeplogonew.png",
    apple: "/metronic/upload/aeplogonew.png",
  },
};

export default function RootLayout({
  children,
}: {
  children: ReactNode;
}) {
  return (
    <html lang="id">
      <head>
        <link
          rel="stylesheet"
          href="/metronic/assets/plugins/global/plugins.bundle.css"
        />
        <link
          rel="stylesheet"
          href="/metronic/assets/css/style.bundle.css"
        />
      </head>
      <body
        id="kt_body"
        data-kt-name="metronic"
        className="header-tablet-and-mobile-fixed aside-enabled"
      >
        {children}
        <Script
          src="/metronic/assets/plugins/global/plugins.bundle.js"
          strategy="afterInteractive"
        />
      </body>
    </html>
  );
}
