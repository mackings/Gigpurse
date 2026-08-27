import Link from "next/link";
import { Card } from "@/components/ui/card";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { formatMoney, postedAgo } from "@/lib/utils";
import { Handshake, Briefcase, ArrowRight } from "lucide-react";

const STATUS_COLOR = {
  active: "bg-primary",
  completed: "bg-status-success",
  disputed: "bg-status-critical",
  cancelled: "bg-status-critical",
};

// Shared contract list card for both the client and talent dashboards, so
// "Active contracts" / "Your contracts" read consistently everywhere.
export default function ContractRow({ contract }) {
  const SourceIcon = contract.source === "direct_hire" ? Handshake : Briefcase;

  return (
    <Link href={`/contracts/${contract.id}`} className="group block h-full">
      <Card className="h-full transition-colors duration-200 hover:border-foreground/15">
        <div className="px-(--card-spacing) flex items-start gap-4">
          <IconBadge icon={SourceIcon} color={STATUS_COLOR[contract.status] || "bg-primary"} />
          <div className="min-w-0 flex-1">
            <div className="flex items-start justify-between gap-3">
              <h3 className="font-semibold text-foreground truncate group-hover:text-primary transition-colors">
                {contract.title || "Contract"}
              </h3>
              <StatusBadge status={contract.status} />
            </div>

            {contract.description && <p className="text-sm text-muted-foreground line-clamp-2 mt-1">{contract.description}</p>}

            {postedAgo(contract.created_at, "Started") && (
              <p className="text-xs text-muted-foreground mt-2">{postedAgo(contract.created_at, "Started")}</p>
            )}

            <div className="flex items-center justify-between mt-3 pt-3 border-t border-border/60">
              <span className="text-base font-semibold text-foreground">{formatMoney(contract.price)}</span>
              <span className="flex items-center gap-1 text-xs font-medium text-primary">
                View contract
                <ArrowRight className="w-3 h-3 group-hover:translate-x-0.5 transition-transform" />
              </span>
            </div>
          </div>
        </div>
      </Card>
    </Link>
  );
}
