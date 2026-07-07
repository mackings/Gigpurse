"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { apiPost } from "@/lib/api";

export default function JobApplyModal({ job, trigger, onApplied }) {
  const [open, setOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [form, setForm] = useState({ proposal: "", price_bid: "" });

  async function handleSubmit(e) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await apiPost("/jobs/apply", {
        job_id: job.id,
        proposal: form.proposal,
        price_bid: parseFloat(form.price_bid) || 0,
      });
      toast.success("Application submitted!");
      setOpen(false);
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
            <Label htmlFor="price_bid">Your price bid</Label>
            <Input
              id="price_bid"
              type="number"
              required
              min="0"
              value={form.price_bid}
              onChange={(e) => setForm({ ...form, price_bid: e.target.value })}
              className="mt-1.5"
            />
          </div>
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
