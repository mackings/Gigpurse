import { cn } from "@/lib/utils";

const SIZES = {
  sm: { box: "w-8 h-8 rounded-lg", icon: "w-4 h-4" },
  md: { box: "w-10 h-10 rounded-xl", icon: "w-5 h-5" },
  lg: { box: "w-12 h-12 rounded-xl", icon: "w-6 h-6" },
};

// Colored icon chip used to give list rows (jobs, contracts, bookings,
// reviews...) a scannable identity instead of plain text-only cards.
export default function IconBadge({ icon: Icon, color = "bg-primary", size = "md", className }) {
  const s = SIZES[size];
  return (
    <div className={cn("flex items-center justify-center shrink-0 shadow-sm", s.box, color, className)}>
      <Icon className={cn(s.icon, "text-white")} />
    </div>
  );
}
