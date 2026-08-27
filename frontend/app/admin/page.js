"use client";

import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import StatCard from "@/components/dashboard/StatCard";
import AreaTrendChart from "@/components/admin/AreaTrendChart";
import BreakdownBars from "@/components/admin/BreakdownBars";
import { formatMoney } from "@/lib/utils";
import { roleCssVar } from "@/lib/role-colors";
import {
  Loader2,
  Users,
  TrendingUp,
  Wallet,
  Lock,
  ShieldAlert,
} from "lucide-react";

// Mirrors components/ui/status-badge.jsx's semantic mapping (same
// --status-* tokens) so a status means the same color everywhere in the
// app, not just on this dashboard.
const STATUS_COLOR = {
  open: "var(--status-info)",
  pending_funding: "var(--status-warning)",
  pending: "var(--status-warning)",
  pending_hire_funding: "var(--status-warning)",
  proposed: "var(--status-warning)",
  active: "var(--primary)",
  accepted: "var(--primary)",
  funded: "var(--status-accent)",
  refunded: "var(--status-accent)",
  completed: "var(--status-success)",
  released: "var(--status-success)",
  resolved: "var(--status-success)",
  rejected: "var(--status-critical)",
  cancelled: "var(--status-critical)",
  disputed: "var(--status-critical)",
  closed: "var(--muted-foreground)",
};
const statusColor = (status) =>
  STATUS_COLOR[status] || "var(--muted-foreground)";

const ROLE_LABEL = {
  musician: "Talent",
  client: "Clients",
  moderator: "Moderators",
  admin: "Admins",
};

function toStatusItems(byStatus) {
  return Object.entries(byStatus || {}).map(([key, value]) => ({
    key,
    label: key.replace(/_/g, " "),
    value,
    color: statusColor(key),
  }));
}

function Section({ title, subtitle, children }) {
  return (
    <div className="bg-card rounded-xl border border-border p-6">
      <h2 className="font-semibold text-foreground">{title}</h2>
      {subtitle && (
        <p className="text-xs text-muted-foreground mt-0.5">{subtitle}</p>
      )}
      <div className="mt-5">{children}</div>
    </div>
  );
}

export default function AdminOverview() {
  const { data, isLoading } = useQuery({
    queryKey: ["admin-analytics"],
    queryFn: () => apiGet("/admin/analytics"),
  });

  if (isLoading) {
    return (
      <div className="flex justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  const roleItems = Object.entries(data?.users_by_role || {}).map(
    ([key, value]) => ({
      key,
      label: ROLE_LABEL[key] || key,
      value,
      color: roleCssVar(key),
    }),
  );

  return (
    <div className="space-y-4">
      <div className="grid sm:grid-cols-2 lg:grid-cols-5 gap-4">
        <StatCard
          icon={TrendingUp}
          label="Platform revenue"
          value={formatMoney(data?.total_platform_revenue ?? 0)}
          color="bg-primary"
        />
        <StatCard
          icon={Wallet}
          label="Total GMV"
          value={formatMoney(data?.total_gmv ?? 0)}
          color="bg-status-info"
        />
        <StatCard
          icon={Lock}
          label="Held in escrow"
          value={formatMoney(data?.total_held_in_escrow ?? 0)}
          color="bg-status-accent"
        />
        <StatCard
          icon={Users}
          label="Total users"
          value={data?.total_users ?? 0}
          color="bg-status-success"
        />
        <StatCard
          icon={ShieldAlert}
          label="Open disputes"
          value={data?.disputes_by_status?.open ?? 0}
          color="bg-status-critical"
        />
      </div>

      <div className="grid lg:grid-cols-2 gap-4">
        <Section
          title="Platform revenue"
          subtitle="Commission + service fee, last 30 days"
        >
          <AreaTrendChart data={data?.revenue_time_series} money />
        </Section>
        <Section title="New signups" subtitle="All roles, last 30 days">
          <AreaTrendChart data={data?.signups_time_series} />
        </Section>
      </div>

      <div className="grid lg:grid-cols-2 gap-4">
        <Section title="Users by role">
          <BreakdownBars items={roleItems} />
        </Section>
        <Section title="Jobs by status">
          <BreakdownBars items={toStatusItems(data?.jobs_by_status)} />
        </Section>
        <Section title="Contracts by status">
          <BreakdownBars items={toStatusItems(data?.contracts_by_status)} />
        </Section>
        <Section title="Disputes by status">
          <BreakdownBars items={toStatusItems(data?.disputes_by_status)} />
        </Section>
      </div>

      <div className="grid sm:grid-cols-3 gap-4">
        <StatCard
          icon={Users}
          label="New users (7 days)"
          value={data?.new_users_last_7_days ?? 0}
          color="bg-primary"
        />
        <StatCard
          icon={Users}
          label="New users (30 days)"
          value={data?.new_users_last_30_days ?? 0}
          color="bg-status-info"
        />
        <StatCard
          icon={Wallet}
          label="Total contracts"
          value={data?.total_contracts ?? 0}
          color="bg-status-success"
        />
      </div>
    </div>
  );
}
