"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiDelete } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Loader2, Trash2 } from "lucide-react";
import { toast } from "sonner";

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
          <div key={job.id} className="bg-card rounded-xl border border-border p-4 flex items-center justify-between gap-4">
            <div>
              <div className="flex items-center gap-2">
                <p className="font-medium text-foreground">{job.title}</p>
                <Badge variant="secondary" className="capitalize">{job.status}</Badge>
              </div>
              <p className="text-sm text-muted-foreground mt-1">{job.location} · {job.budget}</p>
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
