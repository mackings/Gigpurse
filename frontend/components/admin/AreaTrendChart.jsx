"use client";

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { formatMoney } from "@/lib/utils";

function formatDateShort(dateStr) {
  const d = new Date(dateStr + "T00:00:00");
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

// Value leads, label follows (per dataviz skill) — the number is the
// high-contrast element, the date is secondary.
function ChartTooltip({ active, payload, label, money }) {
  if (!active || !payload?.length) return null;
  const value = payload[0].value;
  return (
    <div className="bg-popover border border-border rounded-lg shadow-lg px-3 py-2 text-sm">
      <p className="font-semibold text-foreground tabular-nums">
        {money ? formatMoney(value) : value.toLocaleString()}
      </p>
      <p className="text-xs text-muted-foreground">{formatDateShort(label)}</p>
    </div>
  );
}

// Single-series trend (last 30 days) — one hue, area wash at ~10% opacity,
// 2px line, hairline recessive gridlines, crosshair tooltip. No legend
// needed for one series; the card title above this component names it.
export default function AreaTrendChart({
  data,
  money = false,
  emptyLabel = "No data yet",
}) {
  const points = data?.length ? data : [];
  if (!points.length) {
    return (
      <div className="h-[220px] flex items-center justify-center text-sm text-muted-foreground">
        {emptyLabel}
      </div>
    );
  }
  return (
    <div className="h-[220px]">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart
          data={points}
          margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
        >
          <defs>
            <linearGradient id="trendFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="var(--chart-1)" stopOpacity={0.22} />
              <stop offset="100%" stopColor="var(--chart-1)" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid
            vertical={false}
            stroke="var(--border)"
            strokeDasharray="0"
          />
          <XAxis
            dataKey="date"
            tickFormatter={formatDateShort}
            tick={{ fill: "var(--muted-foreground)", fontSize: 11 }}
            axisLine={{ stroke: "var(--border)" }}
            tickLine={false}
            minTickGap={32}
          />
          <YAxis
            tick={{ fill: "var(--muted-foreground)", fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            width={money ? 56 : 32}
            tickFormatter={(v) =>
              money ? (v >= 1000 ? `₦${(v / 1000).toFixed(0)}k` : `₦${v}`) : v
            }
          />
          <Tooltip
            content={<ChartTooltip money={money} />}
            cursor={{ stroke: "var(--border)", strokeWidth: 1 }}
          />
          <Area
            type="monotone"
            dataKey="value"
            stroke="var(--chart-1)"
            strokeWidth={2}
            fill="url(#trendFill)"
            dot={false}
            activeDot={{
              r: 4,
              fill: "var(--chart-1)",
              stroke: "var(--card)",
              strokeWidth: 2,
            }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
