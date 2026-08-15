"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { Button } from "@/components/ui/button";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
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
import { Lock, CheckCircle2, Check, X, RefreshCw, Clock, Flag, Loader2, Undo2, Ban, BellRing, ShieldAlert } from "lucide-react";
import { toast } from "sonner";
import { formatMoney } from "@/lib/utils";
import { openPaymentPopup } from "@/lib/payment-popup";
import MilestoneCounterModal from "@/components/milestones/MilestoneCounterModal";

// GigPurse's cut isn't deducted from the milestone amount shown — it can't
// be (PayPetal has no partial-release), so it rides along as an additional
// charge to the client / a smaller release to the talent. Worth surfacing
// right where "Fund escrow" appears, or the checkout total looks wrong.
function usePricing() {
  const { data } = useQuery({ queryKey: ["pricing"], queryFn: () => apiGet("/pricing"), staleTime: Infinity });
  return {
    talentRate: data?.talent_commission_rate ?? 0.1,
    clientRate: data?.client_service_fee_rate ?? 0.05,
  };
}

const STATUS_ICON = {
  proposed: Clock,
  accepted: Flag,
  funded: Lock,
  released: CheckCircle2,
  rejected: X,
  cancelled: Ban,
  disputed: ShieldAlert,
};

const STATUS_COLOR = {
  proposed: "bg-amber-500",
  accepted: "bg-primary",
  funded: "bg-violet-500",
  released: "bg-emerald-500",
  rejected: "bg-rose-500",
  cancelled: "bg-muted-foreground",
  disputed: "bg-rose-500",
};

export default function MilestoneList({
  milestones,
  role,
  currentUserId,
  onAccept,
  onReject,
  onWithdraw,
  onCounter,
  onFund,
  onRelease,
  onCancel,
  onRequestRelease,
  onRefresh,
}) {
  // Tracks "<milestoneId>:<action>" for whichever single button is mid-request,
  // so only that button shows a spinner — its siblings on the same card are
  // merely disabled (not spinning) to block a double-submit race.
  const [pendingKey, setPendingKey] = useState(null);
  const { talentRate, clientRate } = usePricing();

  if (!milestones.length) {
    return <p className="text-sm text-muted-foreground">No milestones proposed yet.</p>;
  }

  async function run(action, milestone, actionName, successMsg) {
    const key = `${milestone.id}:${actionName}`;
    setPendingKey(key);
    try {
      await action(milestone);
      toast.success(successMsg);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setPendingKey(null);
    }
  }

  // Funding now means paying — open PayPetal's hosted checkout in a popup
  // rather than navigating away, so the client never actually leaves
  // GigPurse. The milestone doesn't flip to "funded" until that payment is
  // confirmed (see FinalizeFund, triggered by the webhook or the popup's own
  // poll) — once the popup closes, re-sync so the new status shows up
  // without the user needing to manually refresh.
  async function handleFund(milestone) {
    const key = `${milestone.id}:fund`;
    setPendingKey(key);
    try {
      const result = await onFund(milestone);
      if (result?.payment_url) {
        openPaymentPopup(result.payment_url, { onClose: () => onRefresh?.(milestone) });
        return;
      }
    } catch (err) {
      if (err.code === "payout_account_required") {
        toast.error("This talent hasn't added a payout account yet — they've been notified to add one.");
      } else {
        toast.error(err.message);
      }
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
                  {m.status === "accepted" && m.dispute_id &&
                    (role === "client"
                      ? ` · you'll pay ${formatMoney(m.amount)}, no platform fee — this is a moderator-ordered settlement — not active until funded`
                      : ` · you'll receive the full ${formatMoney(m.amount)}, no platform fee — pending funding`)}
                  {m.status === "accepted" && !m.dispute_id &&
                    (role === "client"
                      ? ` · you'll pay ${formatMoney(m.amount * (1 + clientRate))} (includes ${Math.round(clientRate * 100)}% platform fee) — not active until funded`
                      : ` · you'll receive ${formatMoney(m.amount * (1 - talentRate))} after ${Math.round(talentRate * 100)}% platform fee — pending funding`)}
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
                    onClick={() => run(onReject, m, "reject", "Milestone rejected.")}
                    className="gap-1.5"
                  >
                    {isPending("reject") ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <X className="w-3.5 h-3.5" />}
                    Reject
                  </Button>
                  <MilestoneCounterModal
                    current={m}
                    onCounter={(terms) => onCounter(m, terms)}
                    trigger={
                      <Button size="sm" variant="outline" disabled={cardPending} className="gap-1.5">
                        <RefreshCw className="w-3.5 h-3.5" />
                        Counter-offer
                      </Button>
                    }
                  />
                  <Button size="sm" disabled={cardPending} onClick={() => run(onAccept, m, "accept", "Milestone accepted.")} className="gap-1.5">
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
                  onClick={() => run(onWithdraw, m, "withdraw", "Milestone withdrawn.")}
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
                  onClick={() => handleFund(m)}
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
                  onClick={() => run(onRelease, m, "release", "Payment released to the Talent.")}
                  className="gap-1.5"
                >
                  {isPending("release") ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <CheckCircle2 className="w-3.5 h-3.5" />}
                  Release payment
                </Button>
              )}
              {role === "musician" && m.status === "funded" && !m.dispute_id && (
                <Button
                  size="sm"
                  variant="outline"
                  disabled={cardPending || !!m.release_requested_at}
                  onClick={() => run(onRequestRelease, m, "request-release", "Release requested — the client has 48h to act before it auto-releases.")}
                  className="gap-1.5"
                  title={m.release_requested_at ? "Already requested — auto-releases if the client doesn't act within 48h" : "Ask the client to release this payment"}
                >
                  {isPending("request-release") ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <BellRing className="w-3.5 h-3.5" />}
                  {m.release_requested_at ? "Release requested" : "Request release"}
                </Button>
              )}
              {(m.status === "accepted" || m.status === "funded") && !m.dispute_id && (
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button size="sm" variant="outline" disabled={cardPending} className="gap-1.5 text-muted-foreground hover:text-destructive">
                      <Ban className="w-3.5 h-3.5" />
                      Cancel
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Cancel this milestone?</AlertDialogTitle>
                      <AlertDialogDescription>
                        {m.status === "funded"
                          ? `'${m.title}' is already funded — cancelling it opens a dispute instead of cancelling outright. A moderator will decide how the ${formatMoney(m.amount)} held in escrow gets settled.`
                          : `'${m.title}' hasn't been funded yet, so this cancels it outright — no dispute, just a notification to the other party.`}
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Never mind</AlertDialogCancel>
                      <AlertDialogAction
                        onClick={() =>
                          run(onCancel, m, "cancel", m.status === "funded" ? "Milestone cancelled — a dispute has been opened." : "Milestone cancelled.")
                        }
                        className="gap-1.5"
                      >
                        {m.status === "funded" ? "Cancel & open dispute" : "Cancel milestone"}
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              )}
              {m.dispute_id && m.status === "disputed" && (
                <span className="text-xs font-medium text-rose-600 dark:text-rose-400 flex items-center gap-1">
                  <ShieldAlert className="w-3.5 h-3.5" />
                  Pending dispute resolution
                </span>
              )}
              {m.dispute_id && m.status === "accepted" && (
                <span className="text-xs font-medium text-amber-600 dark:text-amber-400">Ordered by dispute resolution</span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
