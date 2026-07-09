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
import { toast } from "sonner";

export default function WithdrawModal({ trigger, onWithdraw }) {
  const [open, setOpen] = useState(false);
  const [amount, setAmount] = useState("");
  const [bankAccount, setBankAccount] = useState("");

  async function handleSubmit(e) {
    e.preventDefault();
    try {
      await onWithdraw(parseFloat(amount) || 0);
      toast.success("Withdrawal requested.");
      setOpen(false);
      setAmount("");
      setBankAccount("");
    } catch (err) {
      toast.error(err.message);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Withdraw funds</DialogTitle>
          <DialogDescription>Send available balance to your bank account.</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <Label htmlFor="withdraw-amount">Amount (₦)</Label>
            <CurrencyInput id="withdraw-amount" required value={amount} onChange={setAmount} className="mt-1.5" />
          </div>
          <div>
            <Label htmlFor="bank-account">Bank account number</Label>
            <Input id="bank-account" required value={bankAccount} onChange={(e) => setBankAccount(e.target.value)} className="mt-1.5" />
          </div>
          <DialogFooter>
            <Button type="submit">Withdraw</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
