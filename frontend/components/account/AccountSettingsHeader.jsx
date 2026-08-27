"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { cn } from "@/lib/utils";
import { useCurrentUser } from "@/hooks/use-current-user";

// Shared by app/profile/layout.js (Portfolio, My Jobs, Contracts, Bookings)
// AND app/onboarding/page.js directly — onboarding is a legacy top-level
// route, not nested under /profile, so it can't inherit that layout the
// normal way. Without this duplicated here, a talent landing on the
// "Profile" tab lost the back link and tab bar entirely — a real dead end.
export default function AccountSettingsHeader() {
  const pathname = usePathname();
  const { user } = useCurrentUser();
  const isTalent = user?.role === "musician";

  const tabs = isTalent
    ? [
        { href: "/onboarding", label: "Profile" },
        { href: "/profile/portfolio", label: "Portfolio" },
        { href: "/profile/jobs", label: "My Jobs" },
        { href: "/profile/contracts", label: "Contracts" },
        { href: "/profile/bookings", label: "Bookings" },
      ]
    : [
        { href: "/profile", label: "Profile" },
        { href: "/profile/bookings", label: "Bookings" },
      ];

  return (
    <>
      {/* "/" auto-redirects an authenticated visitor to their real dashboard
          (talent → /jobs, client → /dashboard/client), so this is a
          reliable exit without duplicating that role logic here. */}
      <Link href="/" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-6">
        <ArrowLeft className="w-4 h-4" />
        Back to dashboard
      </Link>

      <div className="mb-8">
        <h1 className="text-2xl font-bold text-foreground tracking-tight">Account settings</h1>
        <p className="text-muted-foreground">Manage your details, portfolio, bookings, and job activity in one place.</p>
      </div>

      {tabs.length > 1 && (
        <div className="flex gap-1 border-b border-border mb-8 overflow-x-auto">
          {tabs.map((tab) => (
            <Link
              key={tab.href}
              href={tab.href}
              className={cn(
                "px-4 py-2.5 text-sm font-medium border-b-2 -mb-px whitespace-nowrap transition-colors",
                pathname === tab.href
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
            >
              {tab.label}
            </Link>
          ))}
        </div>
      )}
    </>
  );
}
