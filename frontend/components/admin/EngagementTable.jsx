"use client";

import { useState } from "react";
import { ArrowUp, ArrowDown, ArrowUpDown, AlertTriangle } from "lucide-react";
import { formatMoney } from "@/lib/utils";

function relativeTime(dateStr) {
  if (!dateStr) return null;
  const diffMs = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diffMs / 86400000);
  if (days < 1) return "Today";
  if (days === 1) return "Yesterday";
  if (days < 30) return `${days} days ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months} month${months > 1 ? "s" : ""} ago`;
  return new Date(dateStr).toLocaleDateString();
}

export function isAtRisk(row, windowDays) {
  if (!row.last_engaged_at) return true;
  const days =
    (Date.now() - new Date(row.last_engaged_at).getTime()) / 86400000;
  return days > windowDays;
}

function SortHeader({ label, sortKey, sort, onSort, className }) {
  const active = sort.key === sortKey;
  const Icon = active
    ? sort.dir === "asc"
      ? ArrowUp
      : ArrowDown
    : ArrowUpDown;
  return (
    <th className={`px-4 py-3 font-medium ${className || ""}`}>
      <button
        onClick={() => onSort(sortKey)}
        className="flex items-center gap-1 hover:text-foreground"
      >
        {label}
        <Icon
          className={`w-3.5 h-3.5 ${active ? "text-foreground" : "text-muted-foreground/50"}`}
        />
      </button>
    </th>
  );
}

// Shared sortable table for the admin Talent/Clients engagement tabs — same
// shape, just different labels/columns (clients get an avg-per-month column,
// talent's financial figure is "earned" vs clients' "spent").
export default function EngagementTable({
  rows,
  windowDays,
  financialLabel,
  showAvgPerMonth,
  emptyLabel,
}) {
  const [sort, setSort] = useState({ key: "last_engaged_at", dir: "desc" });

  function handleSort(key) {
    setSort((prev) =>
      prev.key === key
        ? { key, dir: prev.dir === "asc" ? "desc" : "asc" }
        : { key, dir: "desc" },
    );
  }

  const sorted = [...rows].sort((a, b) => {
    let av = a[sort.key];
    let bv = b[sort.key];
    if (sort.key === "last_engaged_at" || sort.key === "joined_at") {
      av = av ? new Date(av).getTime() : 0;
      bv = bv ? new Date(bv).getTime() : 0;
    }
    if (sort.key === "name") {
      av = (av || "").toLowerCase();
      bv = (bv || "").toLowerCase();
    }
    if (av < bv) return sort.dir === "asc" ? -1 : 1;
    if (av > bv) return sort.dir === "asc" ? 1 : -1;
    return 0;
  });

  if (!rows.length) {
    return (
      <p className="text-center text-sm text-muted-foreground py-24">
        {emptyLabel}
      </p>
    );
  }

  return (
    <div className="bg-card rounded-2xl border border-border overflow-hidden overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-left text-muted-foreground">
            <SortHeader
              label="Name"
              sortKey="name"
              sort={sort}
              onSort={handleSort}
            />
            <SortHeader
              label="Last engaged"
              sortKey="last_engaged_at"
              sort={sort}
              onSort={handleSort}
            />
            <SortHeader
              label={`Engagements (${windowDays}d)`}
              sortKey="engagement_count"
              sort={sort}
              onSort={handleSort}
            />
            {showAvgPerMonth && (
              <SortHeader
                label="Avg/month"
                sortKey="avg_engagement_per_month"
                sort={sort}
                onSort={handleSort}
              />
            )}
            <SortHeader
              label="Gigs"
              sortKey="gigs_count"
              sort={sort}
              onSort={handleSort}
            />
            <SortHeader
              label={financialLabel}
              sortKey="financial_total"
              sort={sort}
              onSort={handleSort}
            />
          </tr>
        </thead>
        <tbody>
          {sorted.map((row) => {
            const atRisk = isAtRisk(row, windowDays);
            return (
              <tr
                key={row.user_id}
                className="border-b border-border last:border-0 transition-colors hover:bg-muted/40"
              >
                <td className="px-4 py-3">
                  <p className="text-foreground font-medium">{row.name}</p>
                  <p className="text-xs text-muted-foreground">{row.email}</p>
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-1.5">
                    <span
                      className={
                        atRisk
                          ? "text-rose-600 dark:text-rose-400 font-medium"
                          : "text-muted-foreground"
                      }
                    >
                      {relativeTime(row.last_engaged_at) || "Never"}
                    </span>
                    {atRisk && (
                      <span
                        className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-rose-500/10 text-rose-600 dark:text-rose-400"
                        title={`No activity in the last ${windowDays} days`}
                      >
                        <AlertTriangle className="w-3 h-3" />
                        At risk
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-4 py-3 text-foreground tabular-nums">
                  {row.engagement_count}
                </td>
                {showAvgPerMonth && (
                  <td className="px-4 py-3 text-foreground tabular-nums">
                    {(row.avg_engagement_per_month ?? 0).toFixed(1)}
                  </td>
                )}
                <td className="px-4 py-3 text-foreground tabular-nums">
                  {row.gigs_count}
                </td>
                <td className="px-4 py-3 text-foreground font-medium tabular-nums">
                  {formatMoney(row.financial_total)}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
