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
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { apiPost } from "@/lib/api";

const emptyForm = { title: "", description: "", location: "", eventDate: "", eventTime: "", price: "" };

// Combines a date-only input with an optional time-of-day input into one
// ISO timestamp. Kept as two separate native inputs rather than a single
// datetime-local — Firefox's combined widget has historically had
// rendering issues, and splitting them sidesteps that entirely.
function combineDateTime(date, time) {
  if (!date) return undefined;
  return new Date(`${date}T${time || "00:00"}`).toISOString();
}

export default function BookingModal({ targetUserId, targetName, trigger, onSent }) {
  const [open, setOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [form, setForm] = useState(emptyForm);

  async function handleSubmit(e) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await apiPost("/direct-hires", {
        target_user_id: targetUserId,
        title: form.title,
        description: form.description,
        location: form.location,
        event_date: combineDateTime(form.eventDate, form.eventTime),
        price: parseFloat(form.price) || 0,
      });
      toast.success("Booking request sent!");
      setOpen(false);
      setForm(emptyForm);
      onSent?.();
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
          <DialogTitle>Request a booking</DialogTitle>
          <DialogDescription>Send {targetName} the details of your event.</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <Label htmlFor="title">Event title</Label>
            <Input
              id="title"
              required
              placeholder="Wedding reception, birthday party..."
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              className="mt-1.5"
            />
          </div>
          <div>
            <Label htmlFor="description">Details</Label>
            <Textarea
              id="description"
              required
              placeholder="What you're looking for..."
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              className="mt-1.5"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="location">Location</Label>
              <Input
                id="location"
                placeholder="Lekki, Lagos"
                value={form.location}
                onChange={(e) => setForm({ ...form, location: e.target.value })}
                className="mt-1.5"
              />
            </div>
            <div>
              <Label htmlFor="eventDate">Event date</Label>
              <Input
                id="eventDate"
                type="date"
                value={form.eventDate}
                onChange={(e) => setForm({ ...form, eventDate: e.target.value })}
                className="mt-1.5"
              />
            </div>
          </div>
          <div>
            <Label htmlFor="eventTime">Time (optional)</Label>
            <Input
              id="eventTime"
              type="time"
              value={form.eventTime}
              disabled={!form.eventDate}
              onChange={(e) => setForm({ ...form, eventTime: e.target.value })}
              className="mt-1.5"
            />
          </div>
          <div>
            <Label htmlFor="price">Offered price (₦, in thousands)</Label>
            <CurrencyInput
              id="price"
              unit="thousands"
              required
              value={form.price}
              onChange={(v) => setForm({ ...form, price: v })}
              className="mt-1.5"
            />
          </div>
          <DialogFooter>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : "Send request"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
