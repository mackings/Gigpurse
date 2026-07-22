"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Loader2, Check } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

// Controlled-only (no built-in trigger) — resolving requires having actually
// read the dispute's chat first, so the only place this opens from is the
// dispute detail page's own "Resolve" button, not a blind list-row action.
export default function ResolveDisputeModal({ dispute, clientName, musicianName, open, onOpenChange, onResolve }) {
  const [winnerId, setWinnerId] = useState("");
  const [resolution, setResolution] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    if (!winnerId) {
      toast.error("Pick who won this dispute.");
      return;
    }
    setIsSubmitting(true);
    try {
      await onResolve(winnerId, resolution);
      toast.success("Dispute resolved — any held escrow has been settled.");
      onOpenChange(false);
      setWinnerId("");
      setResolution("");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Resolve dispute</DialogTitle>
          <DialogDescription>
            Pick a winner — any escrow still held on this contract is automatically released or refunded accordingly.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-2">
            {[
              {
                id: dispute?.client_id,
                label: clientName,
                sub: `Client${dispute?.opened_by_id === dispute?.client_id ? " · Complainant" : " · Respondent"}`,
              },
              {
                id: dispute?.musician_id,
                label: musicianName,
                sub: `Talent${dispute?.opened_by_id === dispute?.musician_id ? " · Complainant" : " · Respondent"}`,
              },
            ].map((party) => (
              <button
                key={party.id}
                type="button"
                onClick={() => setWinnerId(party.id)}
                className={cn(
                  "flex items-center justify-between gap-2 rounded-xl border-2 p-3 text-left transition-colors",
                  winnerId === party.id ? "border-primary bg-primary/5" : "border-border hover:border-primary/40"
                )}
              >
                <div>
                  <p className="font-medium text-foreground text-sm">{party.label}</p>
                  <p className="text-xs text-muted-foreground">{party.sub}</p>
                </div>
                {winnerId === party.id && (
                  <div className="w-5 h-5 rounded-full bg-primary flex items-center justify-center shrink-0">
                    <Check className="w-3 h-3 text-primary-foreground" />
                  </div>
                )}
              </button>
            ))}
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
