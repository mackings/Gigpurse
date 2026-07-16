"use client";

import { useState } from "react";
import Link from "next/link";
import { useQueryClient } from "@tanstack/react-query";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useUserInfo } from "@/hooks/use-user-info";
import { useDirectHires, useDirectHire } from "@/hooks/use-direct-hire";
import { Button } from "@/components/ui/button";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import CounterOfferModal from "@/components/booking/CounterOfferModal";
import BookingModal from "@/components/booking/BookingModal";
import { formatMoney } from "@/lib/utils";
import { ChevronDown, ChevronUp, Check, X, RefreshCw, MapPin, Calendar, CalendarPlus, Handshake, Loader2 } from "lucide-react";
import { toast } from "sonner";

function formatEventDate(iso) {
  if (!iso) return null;
  return new Date(iso).toLocaleString([], { dateStyle: "medium", timeStyle: "short" });
}

export default function BookingRequestPanel({ otherUserId, bookingId }) {
  const { user } = useCurrentUser();
  const otherUser = useUserInfo(otherUserId);
  const queryClient = useQueryClient();
  const [expanded, setExpanded] = useState(true);
  const [pendingAction, setPendingAction] = useState(null);

  // Resolve which booking to show: an explicit id from the URL (arrived via
  // notification), or the latest pending one between us and this chat
  // partner (organically opened chat that happens to have one).
  const { requests } = useDirectHires();
  const resolvedId =
    bookingId ||
    requests.find((r) => (r.client_id === otherUserId || r.musician_id === otherUserId) && r.status === "pending")?.id;

  const { request, accept, decline, counter } = useDirectHire(resolvedId);

  // No pending booking with this partner yet — only a client can kick one
  // off (from right here, or from a talent profile page); talent can only
  // respond to a booking a client sends, not originate one.
  if (!resolvedId || !request) {
    if (!otherUserId) return null;
    if (user?.role !== "client") return null;
    return (
      <div className="border-b border-border bg-muted/30 px-4 py-2.5 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <span className="text-sm text-muted-foreground truncate">No active booking with {otherUser?.name || "this user"}.</span>
        <BookingModal
          targetUserId={otherUserId}
          targetName={otherUser?.name || "them"}
          onSent={() => queryClient.invalidateQueries({ queryKey: ["direct-hires"] })}
          trigger={
            <Button size="sm" variant="outline" className="gap-1.5 shrink-0 self-start sm:self-auto">
              <CalendarPlus className="w-3.5 h-3.5" />
              Propose a booking
            </Button>
          }
        />
      </div>
    );
  }
  if (!user) return null;

  const isProposer = request.proposed_by === user.id;
  const canRespond = request.status === "pending" && !isProposer;

  async function run(action, name, successMsg) {
    setPendingAction(name);
    try {
      await action();
      toast.success(successMsg);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setPendingAction(null);
    }
  }

  return (
    <div className="border-b border-border bg-muted/30">
      <button
        type="button"
        onClick={() => setExpanded((v) => !v)}
        className="w-full flex items-center justify-between px-4 py-2.5 text-sm font-medium text-foreground"
      >
        <span className="flex items-center gap-2">
          Booking request
          <StatusBadge status={request.status} />
        </span>
        {expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
      </button>
      {expanded && (
        <div className="px-4 pb-4 space-y-3">
          <div className="bg-card rounded-xl border border-border p-4 flex items-start gap-3">
            <IconBadge icon={Handshake} color="bg-primary" size="sm" />
            <div className="min-w-0 flex-1">
              <p className="font-medium text-foreground">{request.title}</p>
              <p className="text-sm text-muted-foreground mt-1">{request.description}</p>
              <div className="flex flex-wrap gap-3 mt-2 text-xs text-muted-foreground">
                {request.location && (
                  <span className="flex items-center gap-1">
                    <MapPin className="w-3.5 h-3.5" />
                    {request.location}
                  </span>
                )}
                {request.event_date && (
                  <span className="flex items-center gap-1">
                    <Calendar className="w-3.5 h-3.5" />
                    {formatEventDate(request.event_date)}
                  </span>
                )}
              </div>
              <p className="text-lg font-semibold text-foreground mt-2">{formatMoney(request.price)}</p>
              {request.status === "pending" && (
                <p className="text-xs text-muted-foreground mt-1">
                  {isProposer ? "Waiting for their response." : "They made this offer — accept, decline, or counter."}
                </p>
              )}
              {request.status === "accepted" && request.contract_id && (
                <Link href={`/contracts/${request.contract_id}`} className="text-xs text-primary hover:underline mt-1 inline-block">
                  View contract →
                </Link>
              )}
            </div>
          </div>

          {canRespond && (
            <div className="flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="outline"
                disabled={!!pendingAction}
                onClick={() => run(decline, "decline", "Booking declined.")}
                className="gap-1.5"
              >
                {pendingAction === "decline" ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <X className="w-3.5 h-3.5" />}
                Decline
              </Button>
              <CounterOfferModal
                current={request}
                onCounter={(terms) => counter(terms)}
                trigger={
                  <Button size="sm" variant="outline" disabled={!!pendingAction} className="gap-1.5">
                    <RefreshCw className="w-3.5 h-3.5" />
                    Counter-offer
                  </Button>
                }
              />
              <Button
                size="sm"
                disabled={!!pendingAction}
                onClick={() => run(accept, "accept", "Booking accepted — a contract was created.")}
                className="gap-1.5"
              >
                {pendingAction === "accept" ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Check className="w-3.5 h-3.5" />}
                Accept
              </Button>
            </div>
          )}

          {request.history?.length > 1 && (
            <div className="space-y-1.5">
              <p className="text-xs font-medium text-muted-foreground">Offer history</p>
              {request.history.map((entry, idx) => (
                <p key={idx} className="text-xs text-muted-foreground">
                  {entry.proposed_by === user.id ? "You" : "They"} offered {formatMoney(entry.price)}
                </p>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
