"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import CurrencyInput from "@/components/ui/currency-input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogTrigger,
} from "@/components/ui/dialog";
import MediaThumb from "@/components/portfolio/MediaThumb";
import { useCurrentUser } from "@/hooks/use-current-user";
import { Loader2, Check } from "lucide-react";
import { toast } from "sonner";
import { apiPost } from "@/lib/api";
import { cn } from "@/lib/utils";

export default function JobApplyModal({ job, trigger, onApplied }) {
  const { user } = useCurrentUser();
  const portfolio = user?.musician_profile?.portfolio || [];
  const [open, setOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [form, setForm] = useState({ proposal: "", price_bid: "" });
  const [selectedIds, setSelectedIds] = useState([]);

  function toggleItem(id) {
    setSelectedIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await apiPost("/jobs/apply", {
        job_id: job.id,
        proposal: form.proposal,
        price_bid: parseFloat(form.price_bid) || 0,
        portfolio_item_ids: selectedIds,
      });
      toast.success("Application submitted!");
      setOpen(false);
      setSelectedIds([]);
      onApplied?.();
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Apply to &quot;{job.title}&quot;</DialogTitle>
          <DialogDescription>Send a proposal and your price bid to the client.</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <Label htmlFor="proposal">Your pitch</Label>
            <Textarea
              id="proposal"
              required
              placeholder="Tell the client why you're a great fit..."
              value={form.proposal}
              onChange={(e) => setForm({ ...form, proposal: e.target.value })}
              className="mt-1.5"
            />
          </div>
          <div>
            <Label htmlFor="price_bid">Your price bid (₦, in thousands)</Label>
            <CurrencyInput
              id="price_bid"
              unit="thousands"
              required
              value={form.price_bid}
              onChange={(v) => setForm({ ...form, price_bid: v })}
              className="mt-1.5"
            />
          </div>

          {portfolio.length > 0 && (
            <div>
              <Label>
                Attach portfolio items (optional)
                {selectedIds.length > 0 && <span className="text-muted-foreground font-normal"> · {selectedIds.length} selected</span>}
              </Label>
              <div className="mt-1.5 grid grid-cols-4 gap-2 max-h-48 overflow-y-auto pr-1">
                {portfolio.map((item) => {
                  const selected = selectedIds.includes(item.id);
                  return (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => toggleItem(item.id)}
                      title={item.title}
                      className={cn(
                        "relative aspect-square rounded-lg overflow-hidden border-2 transition-colors",
                        selected ? "border-primary" : "border-transparent hover:border-border"
                      )}
                    >
                      <MediaThumb item={item} className="rounded-none" />
                      {selected && (
                        <div className="absolute inset-0 bg-primary/30 flex items-center justify-center">
                          <div className="w-5 h-5 rounded-full bg-primary flex items-center justify-center">
                            <Check className="w-3 h-3 text-primary-foreground" />
                          </div>
                        </div>
                      )}
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          <DialogFooter>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : "Submit application"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
