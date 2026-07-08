"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { Button } from "@/components/ui/button";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import ResolveDisputeModal from "@/components/disputes/ResolveDisputeModal";
import { Loader2, ShieldAlert } from "lucide-react";

const STATUS_COLOR = { open: "bg-rose-500", resolved: "bg-emerald-500", closed: "bg-muted-foreground" };

export default function AdminDisputes() {
  const [status, setStatus] = useState("open");
  const queryClient = useQueryClient();

  const { data: disputes, isLoading } = useQuery({
    queryKey: ["admin-disputes", status],
    queryFn: () => apiGet(`/admin/disputes?status=${status}`),
  });

  return (
    <div className="space-y-4">
      <Select value={status} onValueChange={setStatus}>
        <SelectTrigger className="w-48">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="open">Open</SelectItem>
          <SelectItem value="resolved">Resolved</SelectItem>
          <SelectItem value="closed">Closed</SelectItem>
        </SelectContent>
      </Select>

      {isLoading ? (
        <div className="flex justify-center py-24">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
        </div>
      ) : disputes?.length ? (
        <div className="space-y-3">
          {disputes.map((d) => (
            <div
              key={d.id}
              className="group bg-card rounded-xl border border-border p-4 flex items-start justify-between gap-4 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-rose-500/30"
            >
              <div className="flex items-start gap-3 min-w-0">
                <IconBadge icon={ShieldAlert} color={STATUS_COLOR[d.status] || "bg-muted-foreground"} size="sm" />
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <p className="font-medium text-foreground">{d.reason}</p>
                    <StatusBadge status={d.status} />
                  </div>
                  <p className="text-sm text-muted-foreground mt-1">Contract: {d.contract_id}</p>
                  {d.resolution && <p className="text-sm text-muted-foreground mt-1">Resolution: {d.resolution}</p>}
                </div>
              </div>
              {d.status === "open" && (
                <ResolveDisputeModal
                  disputeId={d.id}
                  trigger={<Button size="sm" className="shrink-0">Resolve</Button>}
                  onResolved={() => queryClient.invalidateQueries({ queryKey: ["admin-disputes"] })}
                />
              )}
            </div>
          ))}
        </div>
      ) : (
        <p className="text-center text-sm text-muted-foreground py-24">No disputes with this status.</p>
      )}
    </div>
  );
}
