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
import { Loader2, Banknote, Wallet, AlertTriangle, RefreshCw } from "lucide-react";
import { formatMoney } from "@/lib/utils";
import { toast } from "sonner";

const KIND = { talent_payout: "Talent payout", client_refund: "Client refund" };
const EMPTY = {
  due: "Nothing to pay right now.",
  sending: "PayPetal isn't sending anything right now.",
  stopping: "Nothing is being taken off automatic payment.",
  pending: "No releases waiting on PayPetal.",
  paid: "No payouts recorded yet.",
};

// Released escrow lands in GigPurse's own PayPetal wallet. With automatic
// payouts on, GigPurse pays each one from the payout sub-wallet through
// PayPetal; this page shows how much is left in it and anything that got
// stuck. Anything an admin takes over is paid by bank transfer and recorded
// here.
export default function AdminPayouts() {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState("due");
  const [paying, setPaying] = useState(null);
  const [note, setNote] = useState("");
  const [saving, setSaving] = useState(false);
  const [busy, setBusy] = useState(null);

  const { data: payouts, isLoading } = useQuery({
    queryKey: ["admin-payouts", status],
    queryFn: () => apiGet(`/admin/payouts?status=${status}`),
    refetchInterval: status === "sending" || status === "stopping" ? 15000 : false,
  });
  const { data: auto } = useQuery({
    queryKey: ["admin-payouts-auto"],
    queryFn: () => apiGet("/admin/payouts/auto"),
  });

  const total = (payouts || []).reduce((sum, p) => sum + p.amount, 0);
  const autoOn = !!auto?.enabled;

  async function act(path, p, done) {
    setBusy(p.id + path);
    try {
      await apiPost(path, { id: p.id });
      toast.success(done);
      queryClient.invalidateQueries({ queryKey: ["admin-payouts"] });
      queryClient.invalidateQueries({ queryKey: ["admin-payouts-auto"] });
    } catch (err) {
      toast.error(err.message);
    } finally {
      setBusy(null);
    }
  }

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
      <AutoPayoutPanel auto={auto} />

      <div className="flex items-center justify-between gap-3 flex-wrap">
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-52" aria-label="Payout status">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="due">To pay</SelectItem>
            <SelectItem value="sending">Being paid by PayPetal</SelectItem>
            <SelectItem value="stopping">Being taken off automatic</SelectItem>
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
                {p.status === "sending" && !p.auto_error && (
                  <p className="text-xs text-muted-foreground mt-1">PayPetal is sending this to their bank.</p>
                )}
                {p.auto_error && (
                  <p className="text-xs text-amber-700 dark:text-amber-400 mt-1 flex items-start gap-1">
                    <AlertTriangle className="w-3.5 h-3.5 mt-px shrink-0" />
                    {p.auto_error}
                  </p>
                )}
                {p.manual_only && p.status === "due" && (
                  <p className="text-xs text-muted-foreground mt-1">Taken off automatic payment. Send it yourself, then mark it paid.</p>
                )}
              </div>
              <div className="flex items-center gap-2 shrink-0 flex-wrap justify-end">
                <p className="font-semibold text-foreground tabular-nums mr-1">{formatMoney(p.amount, { decimals: 2 })}</p>
                {autoOn && (p.status === "sending" || (p.status === "due" && (p.auto_error || p.manual_only))) && (
                  <Button
                    size="sm"
                    variant="outline"
                    className="gap-1.5"
                    disabled={!!busy}
                    onClick={() => act("/admin/payouts/retry", p, "Sent to PayPetal again.")}
                  >
                    {busy === p.id + "/admin/payouts/retry" ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <RefreshCw className="w-3.5 h-3.5" />}
                    {p.manual_only ? "Pay automatically" : "Retry now"}
                  </Button>
                )}
                {autoOn && !p.manual_only && (p.status === "due" || p.status === "sending") && (
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={!!busy}
                    onClick={() => {
                      if (window.confirm(`Stop paying ${p.user_name || "this person"} automatically? Any money set aside for it goes back to the payout wallet, and you'll send it yourself.`)) {
                        act("/admin/payouts/pay-by-hand", p, "You'll pay this one by hand.");
                      }
                    }}
                  >
                    {busy === p.id + "/admin/payouts/pay-by-hand" && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                    Pay by hand
                  </Button>
                )}
                {p.status === "due" && (!autoOn || p.manual_only) && (
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

// AutoPayoutPanel: where automatic payouts come from, and how to top it up.
function AutoPayoutPanel({ auto }) {
  if (!auto) return null;
  if (!auto.enabled) {
    return (
      <div className="p-4 rounded-xl border border-border bg-card text-sm text-muted-foreground flex items-start gap-3">
        <Wallet className="w-5 h-5 shrink-0 text-muted-foreground" />
        <p>Automatic payouts are off, so send each payout below yourself and mark it paid.</p>
      </div>
    );
  }
  return (
    <div className="p-4 rounded-xl border border-border bg-card flex items-start justify-between gap-4 flex-wrap">
      <div className="flex items-start gap-3 min-w-0">
        <Wallet className="w-5 h-5 shrink-0 text-primary mt-0.5" />
        <div className="min-w-0">
          <p className="font-medium text-foreground">Automatic payouts are on</p>
          {auto.error ? (
            <p className="text-sm text-amber-700 dark:text-amber-400 mt-0.5">{auto.error}</p>
          ) : (
            <p className="text-sm text-muted-foreground mt-0.5">
              Top up the payout wallet by bank transfer to {auto.bank_name} · {auto.account_number} · {auto.account_name}. Each payout also
              costs PayPetal&apos;s transfer fee.
            </p>
          )}
        </div>
      </div>
      {!auto.error && (
        <div className="text-right shrink-0">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">Payout wallet</p>
          <p className="text-lg font-semibold text-foreground tabular-nums">{formatMoney(auto.balance, { decimals: 2 })}</p>
        </div>
      )}
    </div>
  );
}
