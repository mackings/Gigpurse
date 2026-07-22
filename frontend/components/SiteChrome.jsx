"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import NavBar from "@/components/NavBar";
import Footer from "@/components/Footer";

const NO_CHROME_ROUTES = ["/role-selection", "/onboarding"];
const PUBLIC_ROUTES_WITH_FOOTER = ["/", "/browse"];

// window.history.length is useless for "is there somewhere to go back to" —
// Chromium seeds an about:blank entry in every fresh tab, so it reads >1
// even on a visitor's very first page load. Track real in-app navigations
// ourselves instead, so pages with a "Back" button (e.g. talent profiles)
// can tell a direct/shared link apart from an actual click-through.
export function hasInAppHistory() {
  if (typeof window === "undefined") return false;
  return parseInt(sessionStorage.getItem("gp_nav_count") || "0", 10) > 1;
}

export default function SiteChrome({ children }) {
  const pathname = usePathname();
  const showChrome = !NO_CHROME_ROUTES.includes(pathname);
  const showFooter =
    PUBLIC_ROUTES_WITH_FOOTER.includes(pathname) || pathname.startsWith("/talent/");

  useEffect(() => {
    const count = parseInt(sessionStorage.getItem("gp_nav_count") || "0", 10);
    sessionStorage.setItem("gp_nav_count", String(count + 1));
  }, [pathname]);

  if (!showChrome) {
    return <>{children}</>;
  }

  return (
    <div className="min-h-screen bg-background flex flex-col">
      <NavBar />
      <main className="flex-1">{children}</main>
      {showFooter && <Footer />}
    </div>
  );
}
