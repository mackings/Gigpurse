"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { useCurrentUser } from "@/hooks/use-current-user";

export default function ProfileLayout({ children }) {
  const pathname = usePathname();
  const { user } = useCurrentUser();
  const isTalent = user?.role === "musician";

  // Bookings apply to both roles (clients send them, talent receive them —
  // and either can propose one now). Portfolio and job bids are talent-only.
  const tabs = isTalent
    ? [
        { href: "/onboarding", label: "Profile" },
        { href: "/profile/portfolio", label: "Portfolio" },
        { href: "/profile/jobs", label: "My Jobs" },
        { href: "/profile/bookings", label: "Bookings" },
      ]
    : [
        { href: "/profile", label: "Profile" },
        { href: "/profile/bookings", label: "Bookings" },
      ];

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-4xl mx-auto px-4 py-12">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-foreground tracking-tight">Account settings</h1>
          <p className="text-muted-foreground">Manage your details, portfolio, bookings, and job activity in one place.</p>
        </div>

        {tabs.length > 1 && (
          <div className="flex gap-1 border-b border-border mb-8">
            {tabs.map((tab) => (
              <Link
                key={tab.href}
                href={tab.href}
                className={cn(
                  "px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors",
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

        {children}
      </div>
    </div>
  );
}
