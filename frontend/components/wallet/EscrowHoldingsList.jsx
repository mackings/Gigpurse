"use client";

import { useState } from "react";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogAction,
  AlertDialogCancel,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { formatMoney } from "@/lib/utils";
import { Briefcase, Flag, Loader2, ShieldCheck, Undo2 } from "lucide-react";
import { toast } from "sonner";

const SCOPE_META = {
  job_hire: { label: "Job hire", icon: Briefcase },
  milestone: { label: "Milestone", icon: Flag },
};

export default function EscrowHoldingsList({ holdings, onRequestRefund }) {
  const [refundingRef, setRefundingRef] = useState(null);

  if (!holdings.length) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        <ShieldCheck className="w-8 h-8 mx-auto mb-2" />
        <p className="text-sm">Nothing currently in escrow.</p>
      </div>
    );
  }

  async function handleRefund(holding) {
    setRefundingRef(holding.reference_id);
    try {
      await onRequestRefund(holding.reference_id);
      toast.success(`Refund requested for '${holding.title}'.`);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setRefundingRef(null);
    }
  }

  return (
    <div className="space-y-2">
      {holdings.map((h) => {
        const meta = SCOPE_META[h.scope_type] || { label: h.scope_type, icon: ShieldCheck };
        const Icon = meta.icon;
        const isRefunding = refundingRef === h.reference_id;
        return (
          <div
            key={h.reference_id}
            className="flex items-center justify-between gap-3 p-3 rounded-xl bg-muted/40 transition-all duration-200 hover:bg-muted hover:shadow-sm"
          >
            <div className="flex items-center gap-3 min-w-0">
              <div className="w-9 h-9 rounded-lg flex items-center justify-center shrink-0 shadow-sm bg-violet-500">
                <Icon className="w-4 h-4 text-white" />
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium text-foreground truncate">{h.title}</p>
                <p className="text-xs text-muted-foreground truncate">
                  {meta.label}
                  {h.counterparty_name && ` · with ${h.counterparty_name}`}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3 shrink-0">
              <p className="text-sm font-semibold text-foreground tabular-nums">{formatMoney(h.amount)}</p>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button size="sm" variant="outline" disabled={isRefunding} className="gap-1.5">
                    {isRefunding ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Undo2 className="w-3.5 h-3.5" />}
                    Refund
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Request a refund?</AlertDialogTitle>
                    <AlertDialogDescription>
                      {formatMoney(h.amount)} held for &apos;{h.title}&apos; will be returned to the account you paid with.{" "}
                      {h.counterparty_name || "The other party"} will be notified.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction onClick={() => handleRefund(h)} className="gap-1.5">
                      Request refund
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          </div>
        );
      })}
    </div>
  );
}
