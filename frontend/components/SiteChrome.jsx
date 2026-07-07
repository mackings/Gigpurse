"use client";

import { usePathname } from "next/navigation";
import NavBar from "@/components/NavBar";
import Footer from "@/components/Footer";

const NO_CHROME_ROUTES = ["/role-selection", "/onboarding"];
const PUBLIC_ROUTES_WITH_FOOTER = ["/", "/browse"];

export default function SiteChrome({ children }) {
  const pathname = usePathname();
  const showChrome = !NO_CHROME_ROUTES.includes(pathname);
  const showFooter =
    PUBLIC_ROUTES_WITH_FOOTER.includes(pathname) || pathname.startsWith("/talent/");

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
