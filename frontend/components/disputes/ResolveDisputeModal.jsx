"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { cn, formatMoney } from "@/lib/utils";

// PayPetal has no partial-release/refund call, so a "pay talent X of Y"
// ruling is: refund the client in full immediately (the only thing PayPetal
// itself supports), then — if X > 0 — a new "Dispute settlement" milestone
// for X is created that the client funds separately. See the backend's
// DisputeUsecase.ResolveDispute for the full mechanics.
//
// Controlled-only (no built-in trigger) — resolving requires having actually
// read the dispute's chat first, so the only place this opens from is the
// dispute detail page's own "Resolve" button, not a blind list-row action.
export default function ResolveDisputeModal({ dispute, clientName, musicianName, open, onOpenChange, onResolve }) {
  const [resolution, setResolution] = useState("");
  const [talentAmount, setTalentAmount] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { data: pricing } = useQuery({ queryKey: ["pricing"], queryFn: () => apiGet("/pricing"), staleTime: Infinity });
  const talentRate = pricing?.talent_commission_rate ?? 0.1;

  const { data: milestones } = useQuery({
    queryKey: ["milestones", dispute?.contract_id],
    queryFn: () => apiGet(`/milestones?contract_id=${dispute.contract_id}`),
    enabled: open && !!dispute?.contract_id,
  });

  // Same scope DisputeUsecase.ResolveDispute uses: one specific milestone if
  // the dispute named one, else every milestone this dispute locked.
  const milestoneId = dispute?.milestone_id;
  const inScope = (milestones || []).filter((m) => m.status === "disputed" && (!milestoneId || m.id === milestoneId));
  // What's actually escrowed per milestone is its agreed price minus
  // GigPurse's commission — mirrors milestoneUsecase.Fund's own math.
  const totalHeld = inScope.reduce((sum, m) => sum + m.amount * (1 - talentRate), 0);

  async function handleSubmit(e) {
    e.preventDefault();
    if (talentAmount === "" || Number(talentAmount) < 0 || Number(talentAmount) > totalHeld) {
      toast.error(`Enter an amount between ${formatMoney(0)} and ${formatMoney(totalHeld)}.`);
      return;
    }
    setIsSubmitting(true);
    try {
      await onResolve(resolution, Number(talentAmount));
      toast.success("Dispute resolved.");
      setResolution("");
      setTalentAmount("");
      onOpenChange(false);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  const isFullRefund = talentAmount !== "" && Number(talentAmount) === 0;
  const isFullRelease = talentAmount !== "" && totalHeld > 0 && Number(talentAmount) === totalHeld;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Resolve dispute</DialogTitle>
          <DialogDescription>
            {formatMoney(totalHeld)} is held for this dispute. Decide how much of it {musicianName || "the talent"} gets —
            the rest goes back to {clientName || "the client"}.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              onClick={() => setTalentAmount("0")}
              className={cn(
                "rounded-xl border-2 p-3 text-left transition-colors",
                isFullRefund ? "border-primary bg-primary/5" : "border-border hover:border-primary/40"
              )}
            >
              <p className="font-medium text-foreground text-sm">Full refund</p>
              <p className="text-xs text-muted-foreground">{clientName || "Client"} keeps everything</p>
            </button>
            <button
              type="button"
              onClick={() => setTalentAmount(String(totalHeld))}
              disabled={totalHeld <= 0}
              className={cn(
                "rounded-xl border-2 p-3 text-left transition-colors disabled:opacity-40 disabled:cursor-not-allowed",
                isFullRelease ? "border-primary bg-primary/5" : "border-border hover:border-primary/40"
              )}
            >
              <p className="font-medium text-foreground text-sm">Full release</p>
              <p className="text-xs text-muted-foreground">{musicianName || "Talent"} keeps everything</p>
            </button>
          </div>

          <div>
            <Label htmlFor="talent-amount">Amount to {musicianName || "the talent"} (₦)</Label>
            <Input
              id="talent-amount"
              type="number"
              min={0}
              max={totalHeld}
              step="0.01"
              required
              placeholder="0"
              value={talentAmount}
              onChange={(e) => setTalentAmount(e.target.value)}
              className="mt-1.5"
            />
            {talentAmount !== "" && !isFullRefund && !isFullRelease && Number(talentAmount) >= 0 && Number(talentAmount) <= totalHeld && (
              <p className="text-xs text-muted-foreground mt-1.5">
                Partial: {clientName || "the client"} is refunded in full now, then separately pays{" "}
                {formatMoney(Number(talentAmount))} to {musicianName || "the talent"} — they&apos;re blocked from posting
                new jobs or hiring until they do.
              </p>
            )}
          </div>

          <div>
            <Label htmlFor="resolution-notes">Resolution notes</Label>
            <Textarea
              id="resolution-notes"
              required
              placeholder="Describe how this was resolved..."
              value={resolution}
              onChange={(e) => setResolution(e.target.value)}
              className="mt-1.5 min-h-[100px]"
            />
          </div>
          <DialogFooter>
            <Button type="submit" disabled={isSubmitting} className="gap-1.5">
              {isSubmitting && <Loader2 className="w-4 h-4 animate-spin" />}
              Resolve dispute
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
