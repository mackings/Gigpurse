"use client";

import { useState } from "react";
import { useQueryClient, useMutation } from "@tanstack/react-query";
import { apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import ApplicantsPanel from "@/components/jobs/ApplicantsPanel";
import { formatMoney } from "@/lib/utils";
import { Briefcase, ChevronRight, Loader2, MapPin, Rocket } from "lucide-react";
import { toast } from "sonner";

const STATUS_COLOR = {
  pending_funding: "bg-amber-500",
  open: "bg-sky-500",
  active: "bg-primary",
  completed: "bg-emerald-500",
  cancelled: "bg-rose-500",
  closed: "bg-muted-foreground",
};

export default function ClientJobCard({ job }) {
  const [panelOpen, setPanelOpen] = useState(false);
  const queryClient = useQueryClient();
  const isDraft = job.status === "pending_funding";

  const goLiveMutation = useMutation({
    mutationFn: () => apiPost("/jobs/fund", { job_id: job.id }),
    onSuccess: () => {
      toast.success("Your gig is now live and open to applicants.");
      queryClient.invalidateQueries({ queryKey: ["client-jobs"] });
    },
    onError: (err) => toast.error(err.message),
  });

  return (
    <>
      <div
        role="button"
        tabIndex={0}
        onClick={() => setPanelOpen(true)}
        onKeyDown={(e) => (e.key === "Enter" || e.key === " ") && setPanelOpen(true)}
        className="group bg-card rounded-2xl border border-border p-5 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30 hover:-translate-y-0.5 cursor-pointer"
      >
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3 min-w-0 flex-1">
            <IconBadge icon={Briefcase} color={STATUS_COLOR[job.status] || "bg-muted-foreground"} />
            <div className="min-w-0 flex-1">
              <h3 className="font-semibold text-foreground truncate" title={job.title}>
                {job.title}
              </h3>
              <div className="flex items-center gap-2 flex-wrap mt-1.5">
                <StatusBadge status={job.status} label={isDraft ? "Draft" : undefined} />
                <span className="text-sm text-muted-foreground flex items-center gap-1.5">
                  <MapPin className="w-3.5 h-3.5 shrink-0" />
                  {job.location} · {formatMoney(job.budget)}
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {isDraft && (
              <Button
                size="sm"
                className="gap-1.5"
                onClick={(e) => {
                  e.stopPropagation();
                  goLiveMutation.mutate();
                }}
                disabled={goLiveMutation.isPending}
              >
                {goLiveMutation.isPending ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Rocket className="w-3.5 h-3.5" />}
                Go live
              </Button>
            )}
            <span className="flex items-center gap-1 text-sm text-muted-foreground group-hover:text-foreground transition-colors">
              {isDraft
                ? "Edit / Delete"
                : job.application_count > 0
                  ? `${job.application_count} applicant${job.application_count === 1 ? "" : "s"}`
                  : "Applicants"}
              <ChevronRight className="w-4 h-4" />
            </span>
          </div>
        </div>
      </div>

      <ApplicantsPanel job={job} open={panelOpen} onOpenChange={setPanelOpen} />
    </>
  );
}
