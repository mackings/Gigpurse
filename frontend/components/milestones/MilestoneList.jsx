"use client";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Lock, CheckCircle2, Check, X, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import MilestoneCounterModal from "@/components/milestones/MilestoneCounterModal";

const statusStyles = {
  proposed: "secondary",
  accepted: "outline",
  rejected: "destructive",
  funded: "outline",
  released: "secondary",
};

export default function MilestoneList({ milestones, role, currentUserId, onAccept, onReject, onCounter, onFund, onRelease }) {
  if (!milestones.length) {
    return <p className="text-sm text-muted-foreground">No milestones proposed yet.</p>;
  }

  async function run(action, id, successMsg) {
    try {
      await action(id);
      toast.success(successMsg);
    } catch (err) {
      toast.error(err.message);
    }
  }

  return (
    <div className="space-y-3">
      {milestones.map((m) => {
        const isProposer = m.proposed_by === currentUserId;
        return (
          <div key={m.id} className="p-4 rounded-xl border border-border bg-card flex items-center justify-between gap-4 flex-wrap">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <p className="font-medium text-foreground truncate">{m.title}</p>
                <Badge variant={statusStyles[m.status] || "outline"} className="capitalize shrink-0">
                  {m.status}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground">
                {m.amount}
                {m.due_date && ` · due ${new Date(m.due_date).toLocaleDateString()}`}
                {m.status === "proposed" && (isProposer ? " · awaiting their response" : " · they proposed this")}
              </p>
            </div>

            <div className="flex items-center gap-2 shrink-0">
              {m.status === "proposed" && !isProposer && (
                <>
                  <Button size="sm" variant="outline" onClick={() => run(onReject, m.id, "Milestone rejected.")} className="gap-1.5">
                    <X className="w-3.5 h-3.5" />
                    Reject
                  </Button>
                  <MilestoneCounterModal
                    current={m}
                    onCounter={(terms) => onCounter(m.id, terms)}
                    trigger={
                      <Button size="sm" variant="outline" className="gap-1.5">
                        <RefreshCw className="w-3.5 h-3.5" />
                        Counter-offer
                      </Button>
                    }
                  />
                  <Button size="sm" onClick={() => run(onAccept, m.id, "Milestone accepted.")} className="gap-1.5">
                    <Check className="w-3.5 h-3.5" />
                    Accept
                  </Button>
                </>
              )}
              {role === "client" && m.status === "accepted" && (
                <Button size="sm" onClick={() => run(onFund, m.id, "Escrow funded for this milestone.")} className="gap-1.5">
                  <Lock className="w-3.5 h-3.5" />
                  Fund escrow
                </Button>
              )}
              {role === "client" && m.status === "funded" && (
                <Button size="sm" onClick={() => run(onRelease, m.id, "Payment released to the Talent.")} className="gap-1.5">
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  Release payment
                </Button>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
