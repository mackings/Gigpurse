"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useDirectHires } from "@/hooks/use-direct-hire";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { formatMoney } from "@/lib/utils";
import { Loader2, MessageCircle, MapPin, Calendar, CalendarClock } from "lucide-react";

const tabs = [
  { value: "all", label: "All" },
  { value: "pending", label: "Pending" },
  { value: "accepted", label: "Accepted" },
  { value: "declined", label: "Declined" },
];

const STATUS_COLOR = {
  pending: "bg-amber-500",
  accepted: "bg-primary",
  declined: "bg-rose-500",
};

function formatEventDate(iso) {
  if (!iso) return null;
  return new Date(iso).toLocaleString([], { dateStyle: "medium", timeStyle: "short" });
}

export default function BookingsPage() {
  const router = useRouter();
  const { user } = useCurrentUser();
  const { requests, isLoading } = useDirectHires();
  const [status, setStatus] = useState("all");

  const filtered = status === "all" ? requests : requests.filter((r) => r.status === status);
  const sorted = [...filtered].sort((a, b) => new Date(b.updated_at) - new Date(a.updated_at));

  return (
    <div>
      <div className="flex gap-2 mb-6">
        {tabs.map((t) => (
          <Button
            key={t.value}
            variant={status === t.value ? "default" : "outline"}
            size="sm"
            onClick={() => setStatus(t.value)}
          >
            {t.label}
          </Button>
        ))}
      </div>

      {isLoading ? (
        <div className="flex justify-center py-24">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
        </div>
      ) : sorted.length ? (
        <div className="space-y-3">
          {sorted.map((req) => {
            const counterpartId = user?.id === req.client_id ? req.musician_id : req.client_id;
            const waitingOnThem = req.status === "pending" && req.proposed_by === user?.id;
            // Once accepted, a booking is backed by a real Contract — clicking
            // it opens the same detail view a job-sourced contract does
            // (milestones, escrow, dispute), just reached from Bookings
            // instead of Contracts. Not a <Link> wrapper since the "View in
            // chat" button inside is itself a link — nested anchors aren't
            // valid HTML.
            const isOpenable = req.status === "accepted" && req.contract_id;
            return (
              <div
                key={req.id}
                role={isOpenable ? "button" : undefined}
                tabIndex={isOpenable ? 0 : undefined}
                onClick={isOpenable ? () => router.push(`/contracts/${req.contract_id}`) : undefined}
                onKeyDown={
                  isOpenable
                    ? (e) => {
                        if (e.key === "Enter") router.push(`/contracts/${req.contract_id}`);
                      }
                    : undefined
                }
                className={`group bg-card rounded-xl border border-border p-4 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30 hover:-translate-y-0.5 ${isOpenable ? "cursor-pointer" : ""}`}
              >
                <div className="flex items-start justify-between gap-3 flex-wrap">
                  <div className="flex items-start gap-3 min-w-0">
                    <IconBadge icon={CalendarClock} color={STATUS_COLOR[req.status] || "bg-muted-foreground"} size="sm" />
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <p className="font-medium text-foreground">{req.title}</p>
                        <StatusBadge status={req.status} />
                        {waitingOnThem && (
                          <Badge variant="outline" className="text-xs">
                            Waiting on them
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground mt-0.5">{req.description}</p>
                      <div className="flex flex-wrap gap-3 mt-2 text-xs text-muted-foreground">
                        {req.location && (
                          <span className="flex items-center gap-1">
                            <MapPin className="w-3.5 h-3.5" />
                            {req.location}
                          </span>
                        )}
                        {req.event_date && (
                          <span className="flex items-center gap-1">
                            <Calendar className="w-3.5 h-3.5" />
                            {formatEventDate(req.event_date)}
                          </span>
                        )}
                      </div>
                      <p className="text-sm font-semibold text-foreground mt-1">{formatMoney(req.price)}</p>
                      {isOpenable && <p className="text-xs text-primary mt-1">Tap to view milestones &amp; escrow</p>}
                    </div>
                  </div>
                  <Link
                    href={`/messages?with=${counterpartId}&booking=${req.id}`}
                    onClick={(e) => e.stopPropagation()}
                    className="shrink-0"
                  >
                    <Button size="sm" variant="outline" className="gap-1.5">
                      <MessageCircle className="w-3.5 h-3.5" />
                      {req.status === "pending" && !waitingOnThem ? "Discuss & respond" : "View in chat"}
                    </Button>
                  </Link>
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <div className="text-center py-24 text-muted-foreground">No {status === "all" ? "" : status} bookings yet.</div>
      )}
    </div>
  );
}
