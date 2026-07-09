"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiDelete } from "@/lib/api";
import { Button } from "@/components/ui/button";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { formatMoney } from "@/lib/utils";
import { Loader2, MapPin, Briefcase, Trash2 } from "lucide-react";
import { toast } from "sonner";

const STATUS_COLOR = {
  open: "bg-sky-500",
  active: "bg-primary",
  completed: "bg-emerald-500",
  cancelled: "bg-rose-500",
};

export default function AdminJobs() {
  const queryClient = useQueryClient();
  const { data: jobs, isLoading } = useQuery({
    queryKey: ["admin-jobs"],
    queryFn: () => apiGet("/admin/jobs"),
  });

  async function handleDelete(jobId) {
    try {
      await apiDelete("/admin/jobs", { job_id: jobId });
      toast.success("Job deleted.");
      queryClient.invalidateQueries({ queryKey: ["admin-jobs"] });
    } catch (err) {
      toast.error(err.message);
    }
  }

  if (isLoading) {
    return (
      <div className="flex justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {jobs?.length ? (
        jobs.map((job) => (
          <div
            key={job.id}
            className="group bg-card rounded-xl border border-border p-4 flex items-center justify-between gap-4 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30"
          >
            <div className="flex items-center gap-3 min-w-0 flex-1">
              <IconBadge icon={Briefcase} color={STATUS_COLOR[job.status] || "bg-muted-foreground"} size="sm" />
              <div className="min-w-0 flex-1">
                <p className="font-medium text-foreground truncate" title={job.title}>
                  {job.title}
                </p>
                <div className="flex items-center gap-2 flex-wrap mt-1">
                  <StatusBadge status={job.status} />
                  <span className="text-sm text-muted-foreground flex items-center gap-1.5">
                    <MapPin className="w-3.5 h-3.5 shrink-0" />
                    {job.location} · {formatMoney(job.budget)}
                  </span>
                </div>
              </div>
            </div>
            <Button size="sm" variant="destructive" onClick={() => handleDelete(job.id)} className="gap-1.5 shrink-0">
              <Trash2 className="w-3.5 h-3.5" />
              Delete
            </Button>
          </div>
        ))
      ) : (
        <p className="text-center text-sm text-muted-foreground py-24">No jobs posted yet.</p>
      )}
    </div>
  );
}
