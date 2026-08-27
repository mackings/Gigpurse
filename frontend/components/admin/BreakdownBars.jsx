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
  const total = rows.reduce((sum, r) => sum + r.value, 0);

  return (
    <div className="space-y-1.5 -mx-2">
      {rows.map((row) => {
        const pct = Math.round((row.value / total) * 100);
        return (
          <div
            key={row.key}
            className="group flex items-center gap-3 px-2 py-2 rounded-lg transition-colors hover:bg-muted/50"
          >
            <span
              className="w-2 h-2 rounded-full shrink-0"
              style={{ backgroundColor: row.color }}
            />
            <span className="w-24 shrink-0 text-sm text-foreground capitalize truncate">
              {row.label}
            </span>
            <div className="flex-1 h-2.5 bg-muted rounded-full overflow-hidden">
              <div
                className="h-full rounded-full transition-all duration-500"
                style={{
                  width: `${Math.max((row.value / max) * 100, 3)}%`,
                  backgroundColor: row.color,
                }}
              />
            </div>
            <span className="w-9 shrink-0 text-xs text-muted-foreground tabular-nums text-right">
              {pct}%
            </span>
            <span className="w-8 shrink-0 text-sm font-semibold text-foreground tabular-nums text-right">
              {row.value}
            </span>
          </div>
        );
      })}
    </div>
  );
}
