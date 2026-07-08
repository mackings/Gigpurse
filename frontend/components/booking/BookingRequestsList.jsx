"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import IconBadge from "@/components/ui/icon-badge";
import { formatMoney } from "@/lib/utils";
import { Loader2, MessageCircle, CalendarClock, MapPin } from "lucide-react";

export default function BookingRequestsList() {
  const { user } = useCurrentUser();
  const { data: requests, isLoading } = useQuery({
    queryKey: ["direct-hires", "pending"],
    queryFn: () => apiGet("/direct-hires?status=pending"),
  });

  if (isLoading) {
    return (
      <div className="flex justify-center py-6">
        <Loader2 className="w-5 h-5 animate-spin text-primary" />
      </div>
    );
  }

  if (!requests?.length) {
    return <p className="text-sm text-muted-foreground">No pending booking requests.</p>;
  }

  return (
    <div className="space-y-3">
      {requests.map((req) => {
        const counterpartId = user?.id === req.client_id ? req.musician_id : req.client_id;
        const waitingOnThem = req.proposed_by === user?.id;
        return (
          <div
            key={req.id}
            className="group bg-card rounded-xl border border-border p-4 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30 hover:-translate-y-0.5"
          >
            <div className="flex items-start justify-between gap-3 flex-wrap">
              <div className="flex items-start gap-3 min-w-0">
                <IconBadge icon={CalendarClock} color={waitingOnThem ? "bg-amber-500" : "bg-primary"} size="sm" />
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <p className="font-medium text-foreground">{req.title}</p>
                    {waitingOnThem && (
                      <Badge variant="outline" className="text-xs">
                        Waiting on them
                      </Badge>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground mt-0.5">{req.description}</p>
                  {req.location && (
                    <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
                      <MapPin className="w-3 h-3 shrink-0" />
                      {req.location}
                    </p>
                  )}
                  <p className="text-sm font-semibold text-foreground mt-1">{formatMoney(req.price)}</p>
                </div>
              </div>
              <Link href={`/messages?with=${counterpartId}&booking=${req.id}`} className="shrink-0">
                <Button size="sm" className="gap-1.5">
                  <MessageCircle className="w-3.5 h-3.5" />
                  {waitingOnThem ? "View" : "Discuss & respond"}
                </Button>
              </Link>
            </div>
          </div>
        );
      })}
    </div>
  );
}
