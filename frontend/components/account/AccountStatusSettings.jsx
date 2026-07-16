"use client";

import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { apiPut } from "@/lib/api";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog";
import { EyeOff, Ban } from "lucide-react";
import { toast } from "sonner";

// Both toggles are self-service and fully reversible — this is a "pause my
// visibility" and "pause my account" pair of switches, not a moderation
// tool, so applying them immediately on toggle (no separate Save step)
// matches how the rest of the app treats this kind of setting.
export default function AccountStatusSettings({ user }) {
  const queryClient = useQueryClient();
  const [confirmDisable, setConfirmDisable] = useState(false);
  const [saving, setSaving] = useState(null);

  async function update(patch) {
    setSaving(Object.keys(patch)[0]);
    try {
      await apiPut("/users/account-status", {
        hide_presence: user.hide_presence ?? false,
        disabled: user.disabled ?? false,
        ...patch,
      });
      queryClient.invalidateQueries({ queryKey: ["profile", "me"] });
      toast.success("Account status updated.");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setSaving(null);
    }
  }

  return (
    <div className="bg-card rounded-2xl border border-border p-6 space-y-5">
      <div>
        <h2 className="font-semibold text-foreground">Account status</h2>
        <p className="text-sm text-muted-foreground mt-0.5">Control how you appear to others — both settings can be reversed anytime.</p>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="flex items-start gap-3">
          <EyeOff className="w-4 h-4 text-muted-foreground mt-0.5 shrink-0" />
          <div>
            <Label htmlFor="hide-presence" className="font-medium text-foreground">
              Appear offline
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">You&apos;ll still show up in search and can message as normal — people just won&apos;t see you as online.</p>
          </div>
        </div>
        <Switch
          id="hide-presence"
          checked={!!user.hide_presence}
          disabled={saving === "hide_presence"}
          onCheckedChange={(checked) => update({ hide_presence: checked })}
        />
      </div>

      <div className="flex items-center justify-between gap-4 pt-4 border-t border-border">
        <div className="flex items-start gap-3">
          <Ban className="w-4 h-4 text-muted-foreground mt-0.5 shrink-0" />
          <div>
            <Label htmlFor="disabled" className="font-medium text-foreground">
              Temporarily disable my account
            </Label>
            <p className="text-xs text-muted-foreground mt-0.5">
              You can still log in to turn this back on. While disabled, nobody can message you
              {user.role === "musician" ? ", and you won't appear in talent search." : "."}
            </p>
          </div>
        </div>
        <Switch
          id="disabled"
          checked={!!user.disabled}
          disabled={saving === "disabled"}
          onCheckedChange={(checked) => (checked ? setConfirmDisable(true) : update({ disabled: false }))}
        />
      </div>

      <AlertDialog open={confirmDisable} onOpenChange={setConfirmDisable}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Disable your account?</AlertDialogTitle>
            <AlertDialogDescription>
              Nobody will be able to message you{user.role === "musician" ? " or find you in talent search" : ""} until you
              turn this back off. You can re-enable it anytime from this same page.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                update({ disabled: true });
                setConfirmDisable(false);
              }}
            >
              Disable account
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
