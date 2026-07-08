"use client";

import { useState } from "react";
import Link from "next/link";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useDirectHires } from "@/hooks/use-direct-hire";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Loader2, MessageCircle, MapPin, Calendar } from "lucide-react";

const tabs = [
  { value: "all", label: "All" },
  { value: "pending", label: "Pending" },
  { value: "accepted", label: "Accepted" },
  { value: "declined", label: "Declined" },
];

const statusVariant = {
  pending: "secondary",
  accepted: "outline",
  declined: "destructive",
};

function formatEventDate(iso) {
  if (!iso) return null;
  return new Date(iso).toLocaleString([], { dateStyle: "medium", timeStyle: "short" });
}

export default function BookingsPage() {
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
            return (
              <div key={req.id} className="bg-card rounded-xl border border-border p-4">
                <div className="flex items-start justify-between gap-3 flex-wrap">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <p className="font-medium text-foreground">{req.title}</p>
                      <Badge variant={statusVariant[req.status] || "outline"} className="capitalize">
                        {req.status}
                      </Badge>
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
                    <p className="text-sm font-semibold text-foreground mt-1">{req.price}</p>
                  </div>
                  <Link href={`/messages?with=${counterpartId}&booking=${req.id}`} className="shrink-0">
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
