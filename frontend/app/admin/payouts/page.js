"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Loader2, Banknote } from "lucide-react";
import { formatMoney } from "@/lib/utils";
import { toast } from "sonner";

const KIND = { talent_payout: "Talent payout", client_refund: "Client refund" };
const EMPTY = {
  due: "Nothing to pay right now.",
  pending: "No releases waiting on PayPetal.",
  paid: "No payouts recorded yet.",
};

// Released escrow lands in GigPurse's own PayPetal wallet, and PayPetal has no
// way to send it on automatically — so each payout and refund is a bank
// transfer an admin makes, then records here.
export default function AdminPayouts() {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState("due");
  const [paying, setPaying] = useState(null);
  const [note, setNote] = useState("");
  const [saving, setSaving] = useState(false);

  const { data: payouts, isLoading } = useQuery({
    queryKey: ["admin-payouts", status],
    queryFn: () => apiGet(`/admin/payouts?status=${status}`),
  });

  const total = (payouts || []).reduce((sum, p) => sum + p.amount, 0);

  async function handlePaid(e) {
    e.preventDefault();
    setSaving(true);
    try {
      await apiPost("/admin/payouts/mark-paid", { id: paying.id, note });
      toast.success("Marked as paid.");
      setPaying(null);
      setNote("");
      queryClient.invalidateQueries({ queryKey: ["admin-payouts"] });
    } catch (err) {
      toast.error(err.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-52" aria-label="Payout status">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="due">To pay</SelectItem>
            <SelectItem value="pending">Waiting on PayPetal</SelectItem>
            <SelectItem value="paid">Paid</SelectItem>
          </SelectContent>
        </Select>
        {payouts?.length > 0 && (
          <p className="text-sm text-muted-foreground">
            {payouts.length} · <span className="font-semibold text-foreground tabular-nums">{formatMoney(total)}</span>
          </p>
        )}
      </div>

      {isLoading ? (
        <div className="py-16 flex justify-center">
          <Loader2 className="w-6 h-6 animate-spin text-primary" />
        </div>
      ) : !payouts?.length ? (
        <div className="text-center py-16 text-muted-foreground">
          <Banknote className="w-8 h-8 mx-auto mb-2" />
          <p className="text-sm">{EMPTY[status]}</p>
        </div>
      ) : (
        <div className="space-y-2">
          {payouts.map((p) => (
            <div key={p.id} className="p-4 rounded-xl border border-border bg-card flex items-center justify-between gap-4 flex-wrap">
              <div className="min-w-0">
                <p className="font-medium text-foreground">
                  {p.user_name || "Unknown user"}{" "}
                  <span className="text-xs font-normal text-muted-foreground">{KIND[p.kind] || p.kind}</span>
                </p>
                <p className="text-sm text-muted-foreground truncate">{p.title}</p>
                <p className="text-sm text-foreground mt-1">
                  {p.account
                    ? `${p.account.bank_name} · ${p.account.account_number} · ${p.account.account_name}`
                    : "No bank account on file. Ask them to add one in Wallet."}
                </p>
                {p.note && <p className="text-xs text-muted-foreground mt-1">Transfer reference: {p.note}</p>}
              </div>
              <div className="flex items-center gap-3 shrink-0">
                <p className="font-semibold text-foreground tabular-nums">{formatMoney(p.amount, { decimals: 2 })}</p>
                {p.status === "due" && (
                  <Button size="sm" onClick={() => setPaying(p)} disabled={!p.account}>
                    Mark paid
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      <Dialog
        open={!!paying}
        onOpenChange={(v) => {
          if (!v) {
            setPaying(null);
            setNote("");
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Record this payment</DialogTitle>
            <DialogDescription>
              Only do this after you&apos;ve sent {paying && formatMoney(paying.amount, { decimals: 2 })} to {paying?.user_name}. They&apos;ll
              be told it&apos;s on its way. This can&apos;t be undone.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handlePaid} className="space-y-4">
            <div>
              <Label htmlFor="transfer-ref">Bank transfer reference</Label>
              <Input id="transfer-ref" required value={note} onChange={(e) => setNote(e.target.value)} className="mt-1.5" />
            </div>
            <DialogFooter>
              <Button type="submit" disabled={saving || !note.trim()} className="gap-1.5">
                {saving && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                I&apos;ve sent it
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
