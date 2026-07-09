"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

const emptyRow = () => ({ title: "", amount: "", due_date: "" });

export default function CreateMilestonesModal({ trigger, onCreate }) {
  const [open, setOpen] = useState(false);
  const [rows, setRows] = useState([emptyRow()]);

  function updateRow(idx, patch) {
    setRows((prev) => prev.map((r, i) => (i === idx ? { ...r, ...patch } : r)));
  }

  function addRow() {
    setRows((prev) => [...prev, emptyRow()]);
  }

  function removeRow(idx) {
    setRows((prev) => prev.filter((_, i) => i !== idx));
  }

  async function handleSubmit(e) {
    e.preventDefault();
    const cleaned = rows
      .filter((r) => r.title && r.amount)
      .map((r) => ({
        title: r.title,
        amount: parseFloat(r.amount) || 0,
        due_date: r.due_date ? new Date(r.due_date).toISOString() : undefined,
      }));
    if (cleaned.length === 0) {
      toast.error("Add at least one milestone with a title and amount.");
      return;
    }
    try {
      await onCreate(cleaned);
      setRows([emptyRow()]);
      setOpen(false);
      toast.success("Milestone proposed — waiting on the other party to accept.");
    } catch (err) {
      toast.error(err.message);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Propose a milestone</DialogTitle>
          <DialogDescription>The other party will need to accept it before it can be funded.</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-3 max-h-80 overflow-y-auto pr-1">
            {rows.map((row, idx) => (
              <div key={idx} className="p-3 rounded-lg border border-border space-y-2">
                <div className="flex items-center justify-between">
                  <Label className="text-xs text-muted-foreground">Milestone {idx + 1}</Label>
                  {rows.length > 1 && (
                    <button type="button" onClick={() => removeRow(idx)} className="text-destructive">
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>
                <Input placeholder="Title (e.g. Rehearsal complete)" value={row.title} onChange={(e) => updateRow(idx, { title: e.target.value })} />
                <div className="grid grid-cols-2 gap-2">
                  <CurrencyInput placeholder="Amount (₦)" value={row.amount} onChange={(v) => updateRow(idx, { amount: v })} />
                  <Input type="date" value={row.due_date} onChange={(e) => updateRow(idx, { due_date: e.target.value })} />
                </div>
              </div>
            ))}
          </div>
          <Button type="button" variant="outline" size="sm" onClick={addRow} className="gap-2">
            <Plus className="w-3.5 h-3.5" />
            Add another milestone
          </Button>
          <DialogFooter>
            <Button type="submit">Propose milestone{rows.length > 1 ? "s" : ""}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
