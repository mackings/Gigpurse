"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ChevronDown, ChevronUp, Loader2 } from "lucide-react";
import { toast } from "sonner";

export default function ClientJobCard({ job }) {
  const [expanded, setExpanded] = useState(false);
  const queryClient = useQueryClient();

  const { data: applications, isLoading } = useQuery({
    queryKey: ["job-applications", job.id],
    queryFn: () => apiGet(`/jobs/applications?job_id=${job.id}`),
    enabled: expanded,
  });

  async function accept(applicationId) {
    try {
      await apiPost("/jobs/applications/accept", { application_id: applicationId });
      toast.success("Application accepted!");
      queryClient.invalidateQueries({ queryKey: ["job-applications", job.id] });
      queryClient.invalidateQueries({ queryKey: ["client-jobs"] });
      queryClient.invalidateQueries({ queryKey: ["contracts"] });
    } catch (err) {
      toast.error(err.message);
    }
  }

  return (
    <div className="bg-card rounded-2xl border border-border p-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-foreground">{job.title}</h3>
            <Badge variant={job.status === "open" ? "secondary" : "outline"} className="capitalize">
              {job.status}
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground mt-1">{job.location} · {job.budget}</p>
        </div>
        <Button variant="ghost" size="sm" onClick={() => setExpanded((v) => !v)} className="gap-1 shrink-0">
          Applications
          {expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </Button>
      </div>

      {expanded && (
        <div className="mt-4 pt-4 border-t border-border space-y-3">
          {isLoading ? (
            <Loader2 className="w-5 h-5 animate-spin text-primary" />
          ) : applications?.length ? (
            applications.map((app) => (
              <div key={app.id} className="flex items-center justify-between gap-4 p-3 rounded-lg bg-muted/50">
                <div>
                  <p className="text-sm text-foreground">{app.proposal}</p>
                  <p className="text-xs text-muted-foreground mt-1">Bid: {app.price_bid} · <span className="capitalize">{app.status}</span></p>
                </div>
                {app.status === "pending" && (
                  <Button size="sm" onClick={() => accept(app.id)} className="shrink-0">
                    Accept
                  </Button>
                )}
              </div>
            ))
          ) : (
            <p className="text-sm text-muted-foreground">No applications yet.</p>
          )}
        </div>
      )}
    </div>
  );
}
