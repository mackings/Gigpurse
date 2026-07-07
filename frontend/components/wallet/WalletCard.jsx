"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Wallet as WalletIcon, Plus } from "lucide-react";
import { toast } from "sonner";

export default function WalletCard({ balance, escrowBalance, onDeposit }) {
  const [open, setOpen] = useState(false);
  const [amount, setAmount] = useState("");

  async function handleDeposit(e) {
    e.preventDefault();
    try {
      await onDeposit(parseFloat(amount) || 0);
      toast.success("Funds added.");
      setOpen(false);
      setAmount("");
    } catch (err) {
      toast.error(err.message);
    }
  }

  return (
    <div className="bg-primary rounded-2xl p-6 text-primary-foreground shadow-sm">
      <div className="flex items-center gap-2 mb-4 opacity-80">
        <WalletIcon className="w-5 h-5" />
        <span className="text-sm font-medium">Available balance</span>
      </div>
      <div className="text-4xl font-bold mb-1">{balance?.toFixed?.(2) ?? "0.00"}</div>
      {escrowBalance > 0 && <p className="text-sm opacity-80 mb-5">{escrowBalance.toFixed(2)} held in escrow</p>}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button variant="secondary" className="gap-2 mt-4">
            <Plus className="w-4 h-4" />
            Add funds
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add funds</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleDeposit} className="space-y-4">
            <Input type="number" min="0" required placeholder="Amount" value={amount} onChange={(e) => setAmount(e.target.value)} />
            <DialogFooter>
              <Button type="submit">Add funds</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
