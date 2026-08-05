"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Loader2, CheckCircle2, Landmark } from "lucide-react";
import { toast } from "sonner";

// Two-step flow, deliberately not skippable: PayPetal can't reverse a
// payout sent to the wrong account, so the resolved account holder name
// must be shown back and explicitly confirmed before it's saved.
export default function AddPayoutAccountModal({ trigger }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [bankCode, setBankCode] = useState("");
  const [accountNumber, setAccountNumber] = useState("");
  const [resolvedName, setResolvedName] = useState("");
  const [isValidating, setIsValidating] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const { data: banks, isLoading: banksLoading } = useQuery({
    queryKey: ["payout-banks"],
    queryFn: () => apiGet("/payout-account/banks"),
    enabled: open,
  });

  const selectedBank = banks?.find((b) => b.code === bankCode);

  function reset() {
    setBankCode("");
    setAccountNumber("");
    setResolvedName("");
  }

  async function handleValidate(e) {
    e.preventDefault();
    setIsValidating(true);
    try {
      const { account_name } = await apiPost("/payout-account/validate", {
        bank_code: bankCode,
        account_number: accountNumber,
      });
      setResolvedName(account_name);
    } catch (err) {
      toast.error(err.message || "Couldn't validate that account.");
    } finally {
      setIsValidating(false);
    }
  }

  async function handleConfirm() {
    setIsSaving(true);
    try {
      await apiPost("/payout-account", {
        bank_code: bankCode,
        bank_name: selectedBank?.name || "",
        account_number: accountNumber,
      });
      toast.success("Payout account saved.");
      queryClient.invalidateQueries({ queryKey: ["wallet"] });
      queryClient.invalidateQueries({ queryKey: ["profile", "me"] });
      setOpen(false);
      reset();
    } catch (err) {
      toast.error(err.message || "Couldn't save that account.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        setOpen(v);
        if (!v) reset();
      }}
    >
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Landmark className="w-5 h-5 text-primary" />
            Payout account
          </DialogTitle>
          <DialogDescription>
            Where you get paid when a client releases escrow. We can&apos;t undo a payout sent to the wrong account, so
            we&apos;ll confirm the account holder&apos;s name before saving.
          </DialogDescription>
        </DialogHeader>

        {!resolvedName ? (
          <form onSubmit={handleValidate} className="space-y-4">
            <div>
              <Label htmlFor="payout-bank">Bank</Label>
              <Select value={bankCode} onValueChange={setBankCode}>
                <SelectTrigger id="payout-bank" className="mt-1.5">
                  <SelectValue placeholder={banksLoading ? "Loading banks..." : "Select your bank"} />
                </SelectTrigger>
                <SelectContent>
                  {(banks || []).map((bank) => (
                    <SelectItem key={bank.code} value={bank.code}>
                      {bank.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label htmlFor="payout-account-number">Account number</Label>
              <Input
                id="payout-account-number"
                required
                inputMode="numeric"
                value={accountNumber}
                onChange={(e) => setAccountNumber(e.target.value.replace(/\D/g, ""))}
                className="mt-1.5"
              />
            </div>
            <DialogFooter>
              <Button type="submit" disabled={isValidating || !bankCode || !accountNumber} className="gap-1.5">
                {isValidating && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                Continue
              </Button>
            </DialogFooter>
          </form>
        ) : (
          <div className="space-y-4">
            <div className="rounded-xl border border-border bg-muted/40 p-4 text-center space-y-1.5">
              <CheckCircle2 className="w-6 h-6 text-emerald-500 mx-auto" />
              <p className="text-sm text-muted-foreground">Confirm this is you</p>
              <p className="font-semibold text-foreground text-lg">{resolvedName}</p>
              <p className="text-xs text-muted-foreground">
                {selectedBank?.name} &middot; {accountNumber}
              </p>
            </div>
            <DialogFooter className="gap-2 sm:gap-2">
              <Button type="button" variant="outline" onClick={() => setResolvedName("")} disabled={isSaving}>
                Back
              </Button>
              <Button type="button" onClick={handleConfirm} disabled={isSaving} className="gap-1.5">
                {isSaving && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                Yes, this is me — save it
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
