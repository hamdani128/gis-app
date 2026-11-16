"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import type { ReactNode } from "react";

import Sidebar from "../../components/layout/sidebar";
import Header from "../../components/layout/header";
import Footer from "../../components/layout/footer";
import { UserProfileProvider } from "../../components/auth/user-profile-context";
import type { UserProfile } from "../../components/auth/user-profile-context";

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const idleTimeoutMs = 60 * 60 * 1000;
const sessionActivityKey = "gis.auth.last-activity";
const sessionLogoutKey = "gis.auth.logged-out";

export default function AdminLayout({
  children,
}: {
  children: ReactNode;
}) {
  const router = useRouter();
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const [desktopSidebarMinimized, setDesktopSidebarMinimized] = useState(false);
  const [checkingSession, setCheckingSession] = useState(true);
  const [profile, setProfile] = useState<UserProfile | null>(null);

  useEffect(() => {
    let active = true;
    fetch(`${apiBase}/api/auth/me`, { credentials: "include", cache: "no-store" })
      .then(async (response) => {
        if (!response.ok) throw new Error("Unauthenticated");
        const result = await response.json();
        if (!result?.data) throw new Error("Profile is missing");
        if (active) {
          setProfile(result.data as UserProfile);
          setCheckingSession(false);
        }
      })
      .catch(() => {
        if (active) router.replace("/auth/login");
      });
    return () => { active = false; };
  }, [router]);

  const logout = useCallback(async (idleExpired = false) => {
    try {
      localStorage.setItem(sessionLogoutKey, String(Date.now()));
      localStorage.removeItem(sessionActivityKey);
    } catch {
      // Continue logout if browser storage is unavailable.
    }
    try {
      await fetch(`${apiBase}/api/auth/logout`, {
        method: "POST",
        credentials: "include",
      });
    } catch {
      // The page still returns to login if the API is temporarily unavailable.
    } finally {
      router.replace(idleExpired ? "/auth/login?reason=idle" : "/auth/login");
    }
  }, [router]);

  useEffect(() => {
    if (!profile) return;

    let lastActivity = Number(localStorage.getItem(sessionActivityKey)) || Date.now();
    if (!localStorage.getItem(sessionActivityKey)) {
      localStorage.setItem(sessionActivityKey, String(lastActivity));
    }
    let lastRefresh = 0;
    let refreshing = false;
    let loggingOut = false;
    let lastStoredActivity = lastActivity;
    let idleTimer: ReturnType<typeof setTimeout>;

    const expireSession = () => {
      if (loggingOut) return;
      loggingOut = true;
      void logout(true);
    };

    const scheduleIdleTimeout = () => {
      clearTimeout(idleTimer);
      const remaining = idleTimeoutMs - (Date.now() - lastActivity);
      if (remaining <= 0) {
        expireSession();
        return;
      }
      idleTimer = setTimeout(expireSession, remaining);
    };

    const refreshSession = async () => {
      const now = Date.now();
      if (refreshing || now - lastRefresh < 5 * 60 * 1000 || now - lastActivity >= idleTimeoutMs) return;
      refreshing = true;
      try {
        const response = await fetch(`${apiBase}/api/auth/refresh`, {
          method: "POST",
          credentials: "include",
          cache: "no-store",
        });
        if (response.status === 401) {
          expireSession();
          return;
        }
        if (response.ok) lastRefresh = Date.now();
      } catch {
        // A transient network failure should not end an otherwise active session.
      } finally {
        refreshing = false;
      }
    };

    const handleActivity = () => {
      const now = Date.now();
      if (now - lastActivity >= idleTimeoutMs) {
        expireSession();
        return;
      }
      if (now - lastActivity < 1000) return;
      lastActivity = now;
      if (now - lastStoredActivity >= 1000) {
        lastStoredActivity = now;
        localStorage.setItem(sessionActivityKey, String(now));
      }
      scheduleIdleTimeout();
      void refreshSession();
    };

    const handleStorage = (event: StorageEvent) => {
      if (event.key === sessionLogoutKey && event.newValue) {
        if (!loggingOut) {
          loggingOut = true;
          void logout();
        }
      } else if (event.key === sessionActivityKey && event.newValue) {
        const activity = Number(event.newValue);
        if (Number.isFinite(activity) && activity > lastActivity) {
          lastActivity = activity;
          scheduleIdleTimeout();
        }
      }
    };

    const handleVisibility = () => {
      if (document.visibilityState === "visible") handleActivity();
    };

    if (Date.now() - lastActivity >= idleTimeoutMs) {
      expireSession();
      return () => {};
    }

    scheduleIdleTimeout();
    void refreshSession();
    const activityEvents: Array<keyof WindowEventMap> = ["pointerdown", "pointermove", "keydown", "wheel", "touchstart", "click"];
    activityEvents.forEach((eventName) => window.addEventListener(eventName, handleActivity, { passive: true }));
    window.addEventListener("storage", handleStorage);
    document.addEventListener("visibilitychange", handleVisibility);
    const refreshTimer = setInterval(() => { void refreshSession(); }, 60 * 1000);

    return () => {
      clearTimeout(idleTimer);
      clearInterval(refreshTimer);
      activityEvents.forEach((eventName) => window.removeEventListener(eventName, handleActivity));
      window.removeEventListener("storage", handleStorage);
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [profile, logout]);

  if (checkingSession || !profile) {
    return (
      <div className="d-flex flex-column flex-root min-vh-100 justify-content-center align-items-center">
        <span className="spinner-border text-primary mb-4" role="status" aria-label="Memeriksa sesi" />
        <span className="text-gray-600 fw-semibold">Memeriksa sesi login...</span>
      </div>
    );
  }

  return (
    <UserProfileProvider profile={profile} onUpdate={setProfile}>
      <div
        className="d-flex flex-column flex-root"
        data-kt-aside-minimize={desktopSidebarMinimized ? "on" : undefined}
      >
        <div className="page d-flex flex-row flex-column-fluid">
          <Sidebar
            mobileOpen={mobileSidebarOpen}
            onMobileClose={() => setMobileSidebarOpen(false)}
          />
          {mobileSidebarOpen && (
            <button
              type="button"
              className="mobile-aside-backdrop d-lg-none"
              aria-label="Tutup menu navigasi"
              onClick={() => setMobileSidebarOpen(false)}
            />
          )}

          <div
            className="wrapper d-flex flex-column flex-row-fluid"
            id="kt_wrapper"
          >
            <Header
              mobileSidebarOpen={mobileSidebarOpen}
              desktopSidebarMinimized={desktopSidebarMinimized}
              onToggleDesktopSidebar={() =>
                setDesktopSidebarMinimized((minimized) => !minimized)
              }
              onToggleMobileSidebar={() =>
                setMobileSidebarOpen((open) => !open)
              }
              onLogout={logout}
            />

            <main
              className="content d-flex flex-column flex-column-fluid"
              id="kt_content"
            >
              <div
                className="post d-flex flex-column-fluid"
                id="kt_post"
              >
                <div
                  id="kt_content_container"
                  className="container-fluid px-5 px-xxl-10"
                >
                  {children}
                </div>
              </div>
            </main>

            <Footer />
          </div>
        </div>
      </div>
    </UserProfileProvider>
  );
}
