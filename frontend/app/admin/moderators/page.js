"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Loader2, ShieldCheck, UserPlus, Ban, RotateCcw } from "lucide-react";
import { initials } from "@/lib/utils";
import { toast } from "sonner";

export default function AdminModerators() {
  const queryClient = useQueryClient();
  const [inviteOpen, setInviteOpen] = useState(false);
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [statusPending, setStatusPending] = useState(null);

  const { data: moderators, isLoading } = useQuery({
    queryKey: ["admin-moderators"],
    queryFn: () => apiGet("/admin/moderators"),
  });

  async function handleInvite(e) {
    e.preventDefault();
    setSubmitting(true);
    try {
      await apiPost("/admin/moderators/invite", { email, name });
      toast.success(`Invited ${email} as a moderator.`);
      setInviteOpen(false);
      setEmail("");
      setName("");
      queryClient.invalidateQueries({ queryKey: ["admin-moderators"] });
    } catch (err) {
      toast.error(err.message);
    } finally {
      setSubmitting(false);
    }
  }

  async function handleStatus(userId, active) {
    setStatusPending(userId);
    try {
      await apiPost("/admin/moderators/status", { user_id: userId, active });
      toast.success(
        active ? "Moderator reinstated." : "Moderator access revoked.",
      );
      queryClient.invalidateQueries({ queryKey: ["admin-moderators"] });
    } catch (err) {
      toast.error(err.message);
    } finally {
      setStatusPending(null);
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground max-w-md">
          Moderators can view private dispute conversations and resolve escrow.
          Only admins can grant this — inviting an email is the only way a
          moderator account is created.
        </p>
        <Dialog open={inviteOpen} onOpenChange={setInviteOpen}>
          <Button
            className="gap-2 shrink-0"
            onClick={() => setInviteOpen(true)}
          >
            <UserPlus className="w-4 h-4" />
            Invite moderator
          </Button>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Invite a moderator</DialogTitle>
              <DialogDescription>
                They&apos;ll be able to request a one-time login code at this
                email to start moderating disputes.
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleInvite} className="space-y-3">
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  Email
                </label>
                <Input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="moderator@example.com"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium text-foreground">
                  Name (optional)
                </label>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Defaults to the email's local part"
                />
              </div>
              <DialogFooter>
                <Button type="submit" disabled={submitting} className="gap-2">
                  {submitting && <Loader2 className="w-4 h-4 animate-spin" />}
                  Send invite
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-24">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
        </div>
      ) : moderators?.length ? (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {moderators.map((m) => {
            const active = m.role === "moderator";
            return (
              <div
                key={m.id}
                className="group bg-card rounded-xl border border-border p-5 flex flex-col transition-colors hover:border-foreground/15"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <div
                    className={`w-11 h-11 rounded-full flex items-center justify-center shrink-0 text-sm font-semibold text-white ${active ? "bg-status-success" : "bg-muted-foreground"}`}
                  >
                    {initials(m.name)}
                  </div>
                  <div className="min-w-0">
                    <p className="font-semibold text-foreground truncate">
                      {m.name}
                    </p>
                    <p className="text-sm text-muted-foreground truncate">
                      {m.email}
                    </p>
                  </div>
                </div>
                <span className="inline-flex items-center gap-1 text-xs font-medium text-muted-foreground mt-3">
                  <ShieldCheck className="w-3.5 h-3.5" />
                  Moderator
                </span>

                <div className="flex items-center justify-between gap-3 mt-4 pt-4 border-t border-border">
                  <span
                    className={
                      active
                        ? "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-status-success/10 text-status-success"
                        : "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-muted text-muted-foreground"
                    }
                  >
                    <span className="w-1.5 h-1.5 rounded-full bg-current" />
                    {active ? "Active" : "Revoked"}
                  </span>
                  <Button
                    size="sm"
                    variant={active ? "destructive" : "outline"}
                    disabled={statusPending === m.id}
                    onClick={() => handleStatus(m.id, !active)}
                    className="gap-1.5"
                  >
                    {statusPending === m.id ? (
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    ) : active ? (
                      <Ban className="w-3.5 h-3.5" />
                    ) : (
                      <RotateCcw className="w-3.5 h-3.5" />
                    )}
                    {active ? "Revoke" : "Reinstate"}
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <p className="text-center text-sm text-muted-foreground py-24">
          No moderators yet — invite one to get started.
        </p>
      )}
    </div>
  );
}
