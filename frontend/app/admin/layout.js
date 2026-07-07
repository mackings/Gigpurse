"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { useCurrentUser } from "@/hooks/use-current-user";

const allTabs = [
  { href: "/admin", label: "Overview" },
  { href: "/admin/users", label: "Users" },
  { href: "/admin/jobs", label: "Jobs" },
  { href: "/admin/disputes", label: "Disputes" },
];

export default function AdminLayout({ children }) {
  const pathname = usePathname();
  const { user } = useCurrentUser();
  // Moderators are scoped to disputes only — the other tabs would just
  // bounce them back out via the route guard, so don't show them at all.
  const isModerator = user?.role === "moderator";
  const tabs = isModerator ? allTabs.filter((t) => t.href === "/admin/disputes") : allTabs;

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-6xl mx-auto px-4 py-12">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-foreground tracking-tight">{isModerator ? "Disputes" : "Admin"}</h1>
          <p className="text-muted-foreground">{isModerator ? "Review and resolve reported contracts." : "Platform moderation and analytics."}</p>
        </div>

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

        {children}
      </div>
    </div>
  );
}
