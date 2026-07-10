import { cn } from "@/lib/utils";

const STATUS_STYLES = {
  open: "bg-sky-500/10 text-sky-600 dark:text-sky-400",
  pending: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  proposed: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  waiting: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  active: "bg-primary/10 text-primary",
  accepted: "bg-primary/10 text-primary",
  funded: "bg-violet-500/10 text-violet-600 dark:text-violet-400",
  completed: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  released: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  resolved: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  rejected: "bg-rose-500/10 text-rose-600 dark:text-rose-400",
  cancelled: "bg-rose-500/10 text-rose-600 dark:text-rose-400",
  disputed: "bg-rose-500/10 text-rose-600 dark:text-rose-400",
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
