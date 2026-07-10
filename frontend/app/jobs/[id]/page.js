"use client";

import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { Button } from "@/components/ui/button";
import IconBadge from "@/components/ui/icon-badge";
import SaveJobButton from "@/components/jobs/SaveJobButton";
import JobApplyModal from "@/components/jobs/JobApplyModal";
import { formatMoney } from "@/lib/utils";
import { ArrowLeft, Loader2, MapPin, Banknote, Clock, Music, Guitar, Star, ShieldCheck, CalendarDays } from "lucide-react";

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

  const { data: client } = useQuery({
    queryKey: ["user", job?.client_id],
    queryFn: () => apiGet(`/users/${job.client_id}`),
    enabled: !!job?.client_id,
  });

  const { data: clientReviews } = useQuery({
    queryKey: ["reviews", job?.client_id],
    queryFn: () => apiGet(`/reviews?user_id=${job.client_id}`),
    enabled: !!job?.client_id,
  });

  const { data: clientRating } = useQuery({
    queryKey: ["reviews-average", job?.client_id],
    queryFn: () => apiGet(`/reviews/average?user_id=${job.client_id}`),
    enabled: !!job?.client_id,
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
    return (
      <div className="min-h-screen bg-background flex items-center justify-center text-muted-foreground">Gig not found.</div>
    );
  }

  const isClosed = job.status !== "open";
  const alreadyApplied = (myApplications || []).some((a) => a.job_id === job.id);
  const saved = (savedJobs || []).some((j) => j.id === job.id);

  return (
    <div className="min-h-screen bg-background py-10 px-4">
      <div className="max-w-4xl mx-auto">
        <button
          onClick={() => router.back()}
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground mb-6"
        >
          <ArrowLeft className="w-4 h-4" />
          Back
        </button>

        <div className="grid lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-card rounded-2xl border border-border p-6">
              <div className="flex items-start gap-4">
                <IconBadge icon={Guitar} color={isClosed ? "bg-muted-foreground" : "bg-primary"} size="lg" />
                <div className="min-w-0 flex-1">
                  <div className="flex items-start justify-between gap-3">
                    <h1 className="text-xl font-bold text-foreground">{job.title}</h1>
                    {isMusician && !isClosed && <SaveJobButton jobId={job.id} saved={saved} />}
                  </div>
                  <p className="text-sm text-muted-foreground flex items-center gap-1 mt-1">
                    <Clock className="w-3.5 h-3.5" />
                    Posted {new Date(job.created_at).toLocaleDateString()}
                    {isClosed && <span className="text-rose-500 font-medium ml-1">· No longer available</span>}
                  </p>
                </div>
              </div>

              <div className="flex flex-wrap gap-x-6 gap-y-2 mt-5 pt-5 border-t border-border text-sm">
                <span className="flex items-center gap-1.5 text-foreground font-medium">
                  <Banknote className="w-4 h-4 text-muted-foreground" />
                  {formatMoney(job.budget)}
                </span>
                {job.location && (
                  <span className="flex items-center gap-1.5 text-muted-foreground">
                    <MapPin className="w-4 h-4" />
                    {job.location}
                  </span>
                )}
                {job.instrument && (
                  <span className="flex items-center gap-1.5 text-muted-foreground">
                    <Guitar className="w-4 h-4" />
                    {job.instrument}
                  </span>
                )}
                {job.genre && (
                  <span className="flex items-center gap-1.5 text-muted-foreground">
                    <Music className="w-4 h-4" />
                    {job.genre}
                  </span>
                )}
              </div>
            </div>

            <div className="bg-card rounded-2xl border border-border p-6">
              <h2 className="font-semibold text-foreground mb-3">Summary</h2>
              <p className="text-foreground whitespace-pre-line leading-relaxed">{job.description}</p>
            </div>
          </div>

          <div>
            <div className="bg-card rounded-2xl border border-border p-6 sticky top-24 space-y-4">
              {isMusician ? (
                isClosed ? (
                  <Button disabled className="w-full">
                    No longer available
                  </Button>
                ) : alreadyApplied ? (
                  <Button disabled variant="outline" className="w-full">
                    Applied
                  </Button>
                ) : (
                  <JobApplyModal job={job} trigger={<Button className="w-full">Apply now</Button>} onApplied={onApplied} />
                )
              ) : !isAuthenticated ? (
                <Link href="/login">
                  <Button className="w-full">Log in to apply</Button>
                </Link>
              ) : (
                <p className="text-sm text-muted-foreground">Only talent accounts can apply to gigs.</p>
              )}

              <div className="pt-4 border-t border-border">
                <h3 className="text-sm font-semibold text-foreground mb-3 flex items-center gap-1.5">
                  <ShieldCheck className="w-4 h-4 text-primary" />
                  About the client
                </h3>
                <div className="space-y-2 text-sm">
                  <p className="font-medium text-foreground">{client?.client_profile?.company_name || client?.name || "Client"}</p>
                  {client?.location && (
                    <p className="text-muted-foreground flex items-center gap-1.5">
                      <MapPin className="w-3.5 h-3.5" />
                      {client.location}
                    </p>
                  )}
                  {client?.created_at && (
                    <p className="text-muted-foreground flex items-center gap-1.5">
                      <CalendarDays className="w-3.5 h-3.5" />
                      Member since {new Date(client.created_at).toLocaleDateString(undefined, { year: "numeric", month: "long" })}
                    </p>
                  )}
                  <p className="text-muted-foreground flex items-center gap-1.5">
                    <Star className="w-3.5 h-3.5 text-amber-500 fill-amber-500" />
                    {clientRating?.average_rating ? clientRating.average_rating.toFixed(1) : "New"}
                    <span>· {clientReviews?.length || 0} reviews</span>
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
