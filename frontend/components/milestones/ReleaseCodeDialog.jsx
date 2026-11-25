"use client";

import { useState } from "react";
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
  DialogTrigger,
} from "@/components/ui/dialog";
import { Loader2, Mail } from "lucide-react";
import { toast } from "sonner";

// Money held in PayPetal TrustVault only moves — out to the talent or back to
// the client — once the client enters a 6-digit code PayPetal emails them.
// Two steps: ask for the email, then enter the code.
export default function ReleaseCodeDialog({ trigger, title, description, confirmLabel, successMessage, onSendCode, onConfirm }) {
  const [open, setOpen] = useState(false);
  const [sent, setSent] = useState(false);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(null); // "send" | "confirm"

  function reset() {
    setSent(false);
    setCode("");
  }

  async function handleSend() {
    setBusy("send");
    try {
      await onSendCode();
      setSent(true);
      setCode("");
      toast.success("Code sent. Check your email.");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setBusy(null);
    }
  }

  async function handleConfirm(e) {
    e.preventDefault();
    setBusy("confirm");
    try {
      await onConfirm(code);
      toast.success(successMessage);
      setOpen(false);
      reset();
    } catch (err) {
      toast.error(err.message);
    } finally {
      setBusy(null);
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
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        {!sent ? (
          <DialogFooter>
            <Button type="button" onClick={handleSend} disabled={!!busy} className="gap-1.5">
              {busy === "send" ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Mail className="w-3.5 h-3.5" />}
              Email me the code
            </Button>
          </DialogFooter>
        ) : (
          <form onSubmit={handleConfirm} className="space-y-4">
            <div>
              <Label htmlFor="release-code">6-digit code</Label>
              <Input
                id="release-code"
                required
                autoFocus
                inputMode="numeric"
                autoComplete="one-time-code"
                pattern="\d{6}"
                maxLength={6}
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                className="mt-1.5 tracking-widest tabular-nums"
              />
              <p className="text-xs text-muted-foreground mt-1.5">
                PayPetal emailed it to the address on your account. It works once and expires after 20 minutes.
              </p>
            </div>
            <DialogFooter className="gap-2 sm:gap-2">
              <Button type="button" variant="outline" onClick={handleSend} disabled={!!busy}>
                {busy === "send" ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : "Send a new code"}
              </Button>
              <Button type="submit" disabled={!!busy || code.length !== 6} className="gap-1.5">
                {busy === "confirm" && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                {confirmLabel}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
