"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { ShieldCheck } from "lucide-react";

// Shown once per conversation, right before the very first message goes out
// to a contact — "first time" is derived from the chat history being empty,
// so it naturally never reappears once at least one message exists.
export default function FirstMessageDialog({ open, onOpenChange, onConfirm, recipientName }) {
  const [acknowledged, setAcknowledged] = useState(false);

  function handleConfirm() {
    onConfirm();
    onOpenChange(false);
    setAcknowledged(false);
  }

  function handleOpenChange(next) {
    onOpenChange(next);
    if (!next) setAcknowledged(false);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <div className="w-11 h-11 rounded-xl bg-primary flex items-center justify-center mb-1">
            <ShieldCheck className="w-5 h-5 text-primary-foreground" />
          </div>
          <DialogTitle>Stay safe on GigPurse</DialogTitle>
          <DialogDescription>You&apos;re messaging {recipientName} for the first time. A few quick rules:</DialogDescription>
        </DialogHeader>

        <ul className="text-sm text-muted-foreground space-y-2 list-disc pl-5">
          <li>Don&apos;t share personal contact details or move the conversation off-platform before a contract is in place.</li>
          <li>All payments must go through GigPurse — never pay or accept payment outside the app.</li>
          <li>Be respectful. Harassment or discrimination isn&apos;t tolerated and may lead to account suspension.</li>
        </ul>

        <div className="flex items-start gap-2 mt-1">
          <Checkbox id="ack-safety" checked={acknowledged} onCheckedChange={(c) => setAcknowledged(c === true)} className="mt-0.5" />
          <Label htmlFor="ack-safety" className="text-sm font-normal text-muted-foreground leading-snug">
            I understand and agree to keep payments and communication on GigPurse.
          </Label>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleConfirm} disabled={!acknowledged}>
            Send message
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
