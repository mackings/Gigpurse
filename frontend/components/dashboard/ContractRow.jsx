import Link from "next/link";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { formatMoney } from "@/lib/utils";
import { Handshake, ChevronRight } from "lucide-react";

const STATUS_COLOR = {
  active: "bg-sky-500",
  completed: "bg-emerald-500",
  disputed: "bg-rose-500",
  cancelled: "bg-rose-500",
};

// Shared contract list row for both the client and talent dashboards, so
// "Active contracts" / "Your contracts" read consistently everywhere.
export default function ContractRow({ contract }) {
  return (
    <Link
      href={`/contracts/${contract.id}`}
      className="group bg-card rounded-xl border border-border p-4 flex items-center justify-between gap-3 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30 hover:-translate-y-0.5"
    >
      <div className="flex items-center gap-3 min-w-0">
        <IconBadge icon={Handshake} color={STATUS_COLOR[contract.status] || "bg-primary"} size="sm" />
        <div className="min-w-0">
          <p className="font-medium text-foreground truncate">{contract.title || "Contract"}</p>
          <div className="flex items-center gap-2 mt-0.5">
            <span className="text-sm text-muted-foreground">{formatMoney(contract.price)}</span>
            <StatusBadge status={contract.status} />
          </div>
        </div>
      </div>
      <ChevronRight className="w-4 h-4 text-muted-foreground shrink-0 transition-transform group-hover:translate-x-0.5 group-hover:text-primary" />
    </Link>
  );
}
