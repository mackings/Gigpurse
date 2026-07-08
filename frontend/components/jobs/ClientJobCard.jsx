"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { formatMoney } from "@/lib/utils";
import { Briefcase, ChevronDown, ChevronUp, Loader2, MapPin, User } from "lucide-react";
import { toast } from "sonner";

const STATUS_COLOR = {
  open: "bg-sky-500",
  active: "bg-primary",
  completed: "bg-emerald-500",
  cancelled: "bg-rose-500",
};

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
    <div className="group bg-card rounded-2xl border border-border p-5 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30 hover:-translate-y-0.5">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 min-w-0">
          <IconBadge icon={Briefcase} color={STATUS_COLOR[job.status] || "bg-muted-foreground"} />
          <div className="min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h3 className="font-semibold text-foreground">{job.title}</h3>
              <StatusBadge status={job.status} />
            </div>
            <p className="text-sm text-muted-foreground mt-1 flex items-center gap-1.5">
              <MapPin className="w-3.5 h-3.5 shrink-0" />
              {job.location} · {formatMoney(job.budget)}
            </p>
          </div>
        </div>
        <Button variant="ghost" size="sm" onClick={() => setExpanded((v) => !v)} className="gap-1 shrink-0">
          Applications
          {expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </Button>
      </div>

      {expanded && (
        <div className="mt-4 pt-4 border-t border-border space-y-2">
          {isLoading ? (
            <Loader2 className="w-5 h-5 animate-spin text-primary" />
          ) : applications?.length ? (
            applications.map((app) => (
              <div
                key={app.id}
                className="flex items-center justify-between gap-4 p-3 rounded-xl bg-muted/50 hover:bg-muted transition-colors"
              >
                <div className="flex items-start gap-3 min-w-0">
                  <div className="w-7 h-7 rounded-full bg-accent flex items-center justify-center shrink-0 mt-0.5">
                    <User className="w-3.5 h-3.5 text-accent-foreground" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm text-foreground">{app.proposal}</p>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs font-semibold text-foreground">Bid: {formatMoney(app.price_bid)}</span>
                      <StatusBadge status={app.status} />
                    </div>
                  </div>
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
