"use client";

import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import StatCard from "@/components/dashboard/StatCard";
import AreaTrendChart from "@/components/admin/AreaTrendChart";
import BreakdownBars from "@/components/admin/BreakdownBars";
import { formatMoney } from "@/lib/utils";
import {
  Loader2,
  Users,
  TrendingUp,
  Wallet,
  Lock,
  ShieldAlert,
} from "lucide-react";

// Mirrors components/ui/status-badge.jsx's semantic mapping so a status
// means the same color everywhere in the app, not just on this dashboard.
const STATUS_COLOR = {
  open: "#0ea5e9",
  pending_funding: "#f59e0b",
  pending: "#f59e0b",
  pending_hire_funding: "#f59e0b",
  proposed: "#f59e0b",
  active: "var(--chart-1)",
  accepted: "var(--chart-1)",
  funded: "#8b5cf6",
  refunded: "#8b5cf6",
  completed: "#10b981",
  released: "#10b981",
  resolved: "#10b981",
  rejected: "#f43f5e",
  cancelled: "#f43f5e",
  disputed: "#f43f5e",
  closed: "var(--muted-foreground)",
};
const statusColor = (status) =>
  STATUS_COLOR[status] || "var(--muted-foreground)";

const ROLE_COLOR = {
  musician: "var(--chart-1)",
  client: "var(--chart-2)",
  moderator: "var(--chart-3)",
  admin: "var(--chart-4)",
};
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
    <div className="bg-card rounded-2xl border border-border p-5">
      <h2 className="font-semibold text-foreground">{title}</h2>
      {subtitle && (
        <p className="text-xs text-muted-foreground mt-0.5">{subtitle}</p>
      )}
      <div className="mt-4">{children}</div>
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
      color: ROLE_COLOR[key] || "var(--chart-5)",
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
          color="bg-sky-500"
        />
        <StatCard
          icon={Lock}
          label="Held in escrow"
          value={formatMoney(data?.total_held_in_escrow ?? 0)}
          color="bg-violet-500"
        />
        <StatCard
          icon={Users}
          label="Total users"
          value={data?.total_users ?? 0}
          color="bg-emerald-500"
        />
        <StatCard
          icon={ShieldAlert}
          label="Open disputes"
          value={data?.disputes_by_status?.open ?? 0}
          color="bg-rose-500"
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
          color="bg-sky-500"
        />
        <StatCard
          icon={Wallet}
          label="Total contracts"
          value={data?.total_contracts ?? 0}
          color="bg-emerald-500"
        />
      </div>
    </div>
  );
}
