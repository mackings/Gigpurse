// Part-to-whole breakdown as horizontal bars (per dataviz skill: bar over
// pie for anything meant to be compared, not just eyeballed as a ratio).
// items: [{ key, label, value, color (CSS color string), icon? }]
export default function BreakdownBars({ items, emptyLabel = "No data yet" }) {
  const rows = (items || [])
    .filter((i) => i.value > 0)
    .sort((a, b) => b.value - a.value);
  if (!rows.length) {
    return (
      <p className="text-sm text-muted-foreground py-6 text-center">
        {emptyLabel}
      </p>
    );
  }
  const max = Math.max(...rows.map((r) => r.value));

  return (
    <div className="space-y-3">
      {rows.map((row) => (
        <div key={row.key} className="flex items-center gap-3">
          <span className="w-28 shrink-0 text-sm text-foreground capitalize truncate">
            {row.label}
          </span>
          <div className="flex-1 h-6 bg-muted/40 rounded-sm overflow-hidden">
            <div
              className="h-full rounded-r-sm transition-all duration-500"
              style={{
                width: `${Math.max((row.value / max) * 100, 3)}%`,
                backgroundColor: row.color,
              }}
            />
          </div>
          <span className="w-10 shrink-0 text-sm font-semibold text-foreground tabular-nums text-right">
            {row.value}
          </span>
        </div>
      ))}
    </div>
  );
}
