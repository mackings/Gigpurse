"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetFooter } from "@/components/ui/sheet";
import { Loader2, CheckCircle2, Landmark } from "lucide-react";
import { toast } from "sonner";

// Same two-step validate-then-confirm flow as AddPayoutAccountModal, just in
// the app's bottom-sheet shell instead of a centered dialog — used wherever
// we need to interrupt a flow (like posting a job) to collect this rather
// than send the user off to a settings page and lose their place. Fully
// controlled (open/onOpenChange) so a parent can pop it open automatically
// the moment it detects the account is missing, not just from a click.
export default function AddPayoutAccountSheet({ open, onOpenChange, onSaved }) {
  const queryClient = useQueryClient();
  const [bankCode, setBankCode] = useState("");
  const [accountNumber, setAccountNumber] = useState("");
  const [resolvedName, setResolvedName] = useState("");
  const [isValidating, setIsValidating] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const { data: rawBanks, isLoading: banksLoading } = useQuery({
    queryKey: ["payout-banks"],
    queryFn: () => apiGet("/payout-account/banks"),
    enabled: open,
  });
  // The bank list has occasionally shipped a duplicate code, which crashes
  // React's key uniqueness check in the dropdown below — de-dupe defensively
  // rather than trust the upstream list is clean.
  const banks = rawBanks ? Array.from(new Map(rawBanks.map((b) => [b.code, b])).values()) : rawBanks;

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
      onOpenChange(false);
      reset();
      onSaved?.();
    } catch (err) {
      toast.error(err.message || "Couldn't save that account.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <Sheet
      open={open}
      onOpenChange={(v) => {
        onOpenChange(v);
        if (!v) reset();
      }}
    >
      <SheetContent>
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <Landmark className="w-5 h-5 text-primary" />
            Add a payout account
          </SheetTitle>
          <SheetDescription>
            Where a refund would be paid back to if a dispute on this gig is ever resolved in your favor. We can&apos;t
            undo a payout sent to the wrong account, so we&apos;ll confirm the account holder&apos;s name before saving.
          </SheetDescription>
        </SheetHeader>

        <div className="p-4 sm:p-5 flex-1 overflow-y-auto">
          {!resolvedName ? (
            <form id="payout-sheet-form" onSubmit={handleValidate} className="space-y-4">
              <div>
                <Label htmlFor="payout-sheet-bank">Bank</Label>
                <Select value={bankCode} onValueChange={setBankCode}>
                  <SelectTrigger id="payout-sheet-bank" className="mt-1.5 w-full">
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
                <Label htmlFor="payout-sheet-account-number">Account number</Label>
                <Input
                  id="payout-sheet-account-number"
                  required
                  inputMode="numeric"
                  value={accountNumber}
                  onChange={(e) => setAccountNumber(e.target.value.replace(/\D/g, ""))}
                  className="mt-1.5"
                />
              </div>
            </form>
          ) : (
            <div className="rounded-xl border border-border bg-muted/40 p-4 text-center space-y-1.5">
              <CheckCircle2 className="w-6 h-6 text-status-success mx-auto" />
              <p className="text-sm text-muted-foreground">Confirm this is you</p>
              <p className="font-semibold text-foreground text-lg">{resolvedName}</p>
              <p className="text-xs text-muted-foreground">
                {selectedBank?.name} &middot; {accountNumber}
              </p>
            </div>
          )}
        </div>

        <SheetFooter className="flex-row justify-end gap-2">
          {!resolvedName ? (
            <Button type="submit" form="payout-sheet-form" disabled={isValidating || !bankCode || !accountNumber} className="gap-1.5">
              {isValidating && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
              Continue
            </Button>
          ) : (
            <>
              <Button type="button" variant="outline" onClick={() => setResolvedName("")} disabled={isSaving}>
                Back
              </Button>
              <Button type="button" onClick={handleConfirm} disabled={isSaving} className="gap-1.5">
                {isSaving && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                Yes, this is me — save it
              </Button>
            </>
          )}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
