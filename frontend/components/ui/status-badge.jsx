import { cn } from "@/lib/utils";

// Keyed off the shared --status-* tokens in globals.css (one definition per
// state, light+dark already baked in) instead of hand-picking a Tailwind
// color family per status here — this used to be the single heaviest
// concentration of hardcoded palette colors in the app, and drifted out of
// sync with the separate ROLE_COLOR maps in the admin pages.
const STATUS_STYLES = {
  open: "bg-status-info/10 text-status-info",
  pending_funding: "bg-status-warning/10 text-status-warning",
  pending: "bg-status-warning/10 text-status-warning",
  shortlisted: "bg-status-accent/10 text-status-accent",
  proposed: "bg-status-warning/10 text-status-warning",
  waiting: "bg-status-warning/10 text-status-warning",
  active: "bg-primary/10 text-primary",
  accepted: "bg-primary/10 text-primary",
  funded: "bg-status-accent/10 text-status-accent",
  completed: "bg-status-success/10 text-status-success",
  released: "bg-status-success/10 text-status-success",
  resolved: "bg-status-success/10 text-status-success",
  refunded: "bg-status-accent/10 text-status-accent",
  rejected: "bg-status-critical/10 text-status-critical",
  cancelled: "bg-status-critical/10 text-status-critical",
  disputed: "bg-status-critical/10 text-status-critical",
  closed: "bg-muted text-muted-foreground",
};

// Small colored pill with a dot that matches the status text color, used
// anywhere a job/contract/milestone/dispute status is shown so state reads
// at a glance instead of blending into plain outline badges. Pass `label`
// to show different text than `status` while still keying off status for
// color (e.g. a talent-facing "Closed" pill for a job whose real status is
// "active"/"completed" from the client's side).
export default function StatusBadge({ status, label, className }) {
  const key = (status || "").toLowerCase();
  const style = STATUS_STYLES[key] || "bg-muted text-muted-foreground";
  return (
    <span className={cn("inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium capitalize shrink-0", style, className)}>
      <span className="w-1.5 h-1.5 rounded-full bg-current" />
      {label || status}
    </span>
  );
}
