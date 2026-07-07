"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import ResolveDisputeModal from "@/components/disputes/ResolveDisputeModal";
import { Loader2 } from "lucide-react";

const statusVariant = { open: "secondary", resolved: "outline", closed: "outline" };

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
            <div key={d.id} className="bg-card rounded-xl border border-border p-4 flex items-start justify-between gap-4">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <p className="font-medium text-foreground">{d.reason}</p>
                  <Badge variant={statusVariant[d.status] || "outline"} className="capitalize shrink-0">{d.status}</Badge>
                </div>
                <p className="text-sm text-muted-foreground mt-1">Contract: {d.contract_id}</p>
                {d.resolution && <p className="text-sm text-muted-foreground mt-1">Resolution: {d.resolution}</p>}
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
