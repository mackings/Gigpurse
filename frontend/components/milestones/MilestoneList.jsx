"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { Lock, CheckCircle2, Check, X, RefreshCw, Clock, Flag, Loader2, Undo2 } from "lucide-react";
import { toast } from "sonner";
import { formatMoney } from "@/lib/utils";
import MilestoneCounterModal from "@/components/milestones/MilestoneCounterModal";

const STATUS_ICON = {
  proposed: Clock,
  accepted: Flag,
  funded: Lock,
  released: CheckCircle2,
  rejected: X,
};

const STATUS_COLOR = {
  proposed: "bg-amber-500",
  accepted: "bg-primary",
  funded: "bg-violet-500",
  released: "bg-emerald-500",
  rejected: "bg-rose-500",
};

export default function MilestoneList({ milestones, role, currentUserId, onAccept, onReject, onWithdraw, onCounter, onFund, onRelease }) {
  // Tracks "<milestoneId>:<action>" for whichever single button is mid-request,
  // so only that button shows a spinner — its siblings on the same card are
  // merely disabled (not spinning) to block a double-submit race.
  const [pendingKey, setPendingKey] = useState(null);

  if (!milestones.length) {
    return <p className="text-sm text-muted-foreground">No milestones proposed yet.</p>;
  }

  async function run(action, id, actionName, successMsg) {
    const key = `${id}:${actionName}`;
    setPendingKey(key);
    try {
      await action(id);
      toast.success(successMsg);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setPendingKey(null);
    }
  }

  return (
    <div className="space-y-3">
      {milestones.map((m) => {
        const isProposer = m.proposed_by === currentUserId;
        const StatusIcon = STATUS_ICON[m.status] || Flag;
        const cardPending = pendingKey?.startsWith(`${m.id}:`);
        const isPending = (action) => pendingKey === `${m.id}:${action}`;
        return (
          <div
            key={m.id}
            className="group p-4 rounded-xl border border-border bg-card flex items-center justify-between gap-4 flex-wrap transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30"
          >
            <div className="flex items-start gap-3 min-w-0">
              <IconBadge icon={StatusIcon} color={STATUS_COLOR[m.status] || "bg-muted-foreground"} size="sm" />
              <div className="min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <p className="font-medium text-foreground truncate">{m.title}</p>
                  <StatusBadge status={m.status} />
                </div>
                <p className="text-sm text-muted-foreground">
                  {formatMoney(m.amount)}
                  {m.due_date && ` · due ${new Date(m.due_date).toLocaleDateString()}`}
                  {m.status === "proposed" && (isProposer ? " · awaiting their response" : " · they proposed this")}
                  {m.status === "accepted" && " · pending funding — not active until the client funds escrow"}
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2 shrink-0">
              {m.status === "proposed" && !isProposer && (
                <>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={cardPending}
                    onClick={() => run(onReject, m.id, "reject", "Milestone rejected.")}
                    className="gap-1.5"
                  >
                    {isPending("reject") ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <X className="w-3.5 h-3.5" />}
                    Reject
                  </Button>
                  <MilestoneCounterModal
                    current={m}
                    onCounter={(terms) => onCounter(m.id, terms)}
                    trigger={
                      <Button size="sm" variant="outline" disabled={cardPending} className="gap-1.5">
                        <RefreshCw className="w-3.5 h-3.5" />
                        Counter-offer
                      </Button>
                    }
                  />
                  <Button size="sm" disabled={cardPending} onClick={() => run(onAccept, m.id, "accept", "Milestone accepted.")} className="gap-1.5">
                    {isPending("accept") ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Check className="w-3.5 h-3.5" />}
                    Accept
                  </Button>
                </>
              )}
              {m.status === "proposed" && isProposer && (
                <Button
                  size="sm"
                  variant="outline"
                  disabled={cardPending}
                  onClick={() => run(onWithdraw, m.id, "withdraw", "Milestone withdrawn.")}
                  className="gap-1.5"
                  title="Made a mistake? Withdraw it and send a corrected one."
                >
                  {isPending("withdraw") ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Undo2 className="w-3.5 h-3.5" />}
                  Withdraw
                </Button>
              )}
              {role === "client" && m.status === "accepted" && (
                <Button
                  size="sm"
                  disabled={cardPending}
                  onClick={() => run(onFund, m.id, "fund", "Escrow funded for this milestone.")}
                  className="gap-1.5"
                >
                  {isPending("fund") ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Lock className="w-3.5 h-3.5" />}
                  Fund escrow
                </Button>
              )}
              {role === "client" && m.status === "funded" && (
                <Button
                  size="sm"
                  disabled={cardPending}
                  onClick={() => run(onRelease, m.id, "release", "Payment released to the Talent.")}
                  className="gap-1.5"
                >
                  {isPending("release") ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <CheckCircle2 className="w-3.5 h-3.5" />}
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
