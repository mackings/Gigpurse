"use client";

import { useParams, useRouter } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import JobDetailContent from "@/components/jobs/JobDetailContent";
import { ArrowLeft, Loader2 } from "lucide-react";

export default function JobDetailPage() {
  const { id } = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user, isAuthenticated } = useCurrentUser();
  const isMusician = user?.role === "musician";

  const { data: job, isLoading: jobLoading } = useQuery({
    queryKey: ["job", id],
    queryFn: () => apiGet(`/jobs?id=${id}`),
    enabled: !!id,
  });

  const { data: myApplications } = useQuery({
    queryKey: ["applications", "mine"],
    queryFn: () => apiGet("/jobs/applications"),
    enabled: isMusician,
  });

  const { data: savedJobs } = useQuery({
    queryKey: ["saved-jobs"],
    queryFn: () => apiGet("/jobs/saved"),
    enabled: isMusician,
  });

  const onApplied = () => queryClient.invalidateQueries({ queryKey: ["applications", "mine"] });

  if (jobLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!job) {
    return <div className="min-h-screen bg-background flex items-center justify-center text-muted-foreground">Gig not found.</div>;
  }

  const alreadyApplied = (myApplications || []).some((a) => a.job_id === job.id);
  const saved = (savedJobs || []).some((j) => j.id === job.id);

  return (
    <div className="min-h-screen bg-background py-10 px-4">
      <div className="max-w-2xl mx-auto">
        <button
          onClick={() => router.back()}
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground mb-6"
        >
          <ArrowLeft className="w-4 h-4" />
          Back
        </button>

        <div className="bg-card rounded-2xl border border-border overflow-hidden">
          <JobDetailContent
            job={job}
            currentUser={user}
            isAuthenticated={isAuthenticated}
            alreadyApplied={alreadyApplied}
            saved={saved}
            onApplied={onApplied}
            showOpenInNewWindow={false}
          />
        </div>
      </div>
    </div>
  );
}
