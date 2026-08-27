// Single source for role -> semantic color, replacing two divergent
// ROLE_COLOR maps that used to live independently in app/admin/page.js
// (CSS custom-property values, for chart bars) and app/admin/users/page.js
// (Tailwind utility classes, for a badge pill) — same roles, different
// colors, because each was hand-picked separately. Both now derive from
// this one mapping onto the shared --status-* tokens in globals.css.
const ROLE_TOKEN = {
  musician: "primary",
  client: "info",
  moderator: "success",
  admin: "accent",
};

// For inline chart styles (BreakdownBars, AreaTrendChart) that need an
// actual CSS color value, not a class name.
export function roleCssVar(role) {
  const token = ROLE_TOKEN[role];
  if (!token) return "var(--chart-5)";
  return token === "primary" ? "var(--primary)" : `var(--status-${token})`;
}

// For badge/pill markup that needs a Tailwind class string.
export function roleBadgeClass(role) {
  const token = ROLE_TOKEN[role];
  if (!token) return "bg-muted text-muted-foreground";
  return token === "primary" ? "bg-primary/10 text-primary" : `bg-status-${token}/10 text-status-${token}`;
}
