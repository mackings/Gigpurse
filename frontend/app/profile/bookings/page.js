"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useDirectHires } from "@/hooks/use-direct-hire";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { formatMoney, postedAgo } from "@/lib/utils";
import { Loader2, MessageCircle, MapPin, Calendar, CalendarClock } from "lucide-react";

const tabs = [
  { value: "all", label: "All" },
  { value: "pending", label: "Pending" },
  { value: "accepted", label: "Accepted" },
  { value: "declined", label: "Declined" },
];

const STATUS_COLOR = {
  pending: "bg-status-warning",
  accepted: "bg-primary",
  declined: "bg-status-critical",
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
      <Tabs value={status} onValueChange={setStatus} className="mb-6">
        <TabsList>
          {tabs.map((t) => (
            <TabsTrigger key={t.value} value={t.value}>
              {t.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      {isLoading ? (
        <div className="flex justify-center py-24">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
        </div>
      ) : sorted.length ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
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
              <Card
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
                className={`group h-full transition-colors duration-200 hover:border-foreground/15 ${isOpenable ? "cursor-pointer" : ""}`}
              >
                <div className="px-(--card-spacing) flex flex-col h-full">
                  <div className="flex items-start gap-3">
                    <IconBadge icon={CalendarClock} color={STATUS_COLOR[req.status] || "bg-muted-foreground"} />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-start justify-between gap-2">
                        <p className="font-semibold text-foreground truncate">{req.title}</p>
                        <span className="text-base font-semibold text-foreground shrink-0">{formatMoney(req.price)}</span>
                      </div>
                      <div className="flex items-center gap-2 flex-wrap mt-1">
                        <StatusBadge status={req.status} />
                        {waitingOnThem && (
                          <Badge variant="outline" className="text-xs">
                            Waiting on them
                          </Badge>
                        )}
                      </div>
                    </div>
                  </div>

                  {req.description && <p className="text-sm text-muted-foreground line-clamp-2 mt-3">{req.description}</p>}

                  <div className="flex flex-wrap gap-x-3 gap-y-1 mt-3 text-xs text-muted-foreground">
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
                  {isOpenable && <p className="text-xs text-primary mt-2">Tap to view milestones &amp; escrow</p>}

                  <div className="mt-auto pt-3 flex items-center justify-between gap-2">
                    <span className="text-xs text-muted-foreground">{postedAgo(req.created_at, "Requested")}</span>
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
              </Card>
            );
          })}
        </div>
      ) : (
        <div className="text-center py-24 text-muted-foreground">No {status === "all" ? "" : status} bookings yet.</div>
      )}
    </div>
  );
}
