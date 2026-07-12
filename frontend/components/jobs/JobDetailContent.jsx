"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import SaveJobButton from "@/components/jobs/SaveJobButton";
import JobApplyModal from "@/components/jobs/JobApplyModal";
import {
  formatMoney,
  postedAgo,
  JOB_DURATION_LABELS,
  JOB_EXPERIENCE_LABELS,
  JOB_PROJECT_TYPE_LABELS,
} from "@/lib/utils";
import {
  MapPin,
  Banknote,
  Clock,
  BarChart3,
  Layers,
  ShieldCheck,
  Star,
  CalendarDays,
  Briefcase,
  TrendingUp,
  Link2,
  ExternalLink,
  Check,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";

function InfoCard({ icon: Icon, label, value }) {
  if (!value) return null;
  return (
    <div className="rounded-xl border border-border bg-muted/30 p-3">
      <div className="flex items-center gap-1.5 text-muted-foreground mb-1">
        <Icon className="w-3.5 h-3.5" />
        <span className="text-xs">{label}</span>
      </div>
      <p className="text-sm font-semibold text-foreground">{value}</p>
    </div>
  );
}

function CopyLinkButton({ jobId }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      onClick={() => {
        const url = `${window.location.origin}/jobs/${jobId}`;
        navigator.clipboard.writeText(url).then(() => {
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        });
      }}
      className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
    >
      {copied ? <Check className="w-3.5 h-3.5 text-emerald-500" /> : <Link2 className="w-3.5 h-3.5" />}
      {copied ? "Link copied" : "Copy job link"}
    </button>
  );
}

export default function JobDetailContent({
  job,
  currentUser,
  isAuthenticated,
  alreadyApplied,
  saved,
  onApplied,
  showOpenInNewWindow = false,
}) {
  const queryClient = useQueryClient();
  const isMusician = currentUser?.role === "musician";
  const isOwner = currentUser?.id === job.client_id;
  const isClosed = job.status !== "open";
  const skills = job.skills?.length ? job.skills : [job.instrument, job.genre].filter(Boolean);

  const fundMutation = useMutation({
    mutationFn: () => apiPost("/jobs/fund", { job_id: job.id }),
    onSuccess: () => {
      toast.success("Escrow funded — your gig is now live.");
      queryClient.invalidateQueries({ queryKey: ["job", job.id] });
      queryClient.invalidateQueries({ queryKey: ["client-jobs"] });
    },
    onError: (err) => toast.error(err.message),
  });

  return (
    <div className="flex flex-col overflow-y-auto">
      <div className="p-4 sm:p-5 space-y-6">
        {/* Header */}
        <div>
          <div className="flex items-start justify-between gap-3">
            <h2 className="text-lg font-bold text-foreground leading-snug">{job.title}</h2>
            {isMusician && !isClosed && <SaveJobButton jobId={job.id} saved={saved} />}
          </div>
          <div className="flex items-center gap-3 flex-wrap mt-1.5 text-sm text-muted-foreground">
            {postedAgo(job.created_at) && (
              <span className="flex items-center gap-1">
                <Clock className="w-3.5 h-3.5" />
                {postedAgo(job.created_at)}
              </span>
            )}
            {job.location && (
              <span className="flex items-center gap-1">
                <MapPin className="w-3.5 h-3.5" />
                {job.location}
              </span>
            )}
            {showOpenInNewWindow && (
              <Link
                href={`/jobs/${job.id}`}
                target="_blank"
                className="flex items-center gap-1 text-primary hover:underline"
              >
                Open in a new window
                <ExternalLink className="w-3 h-3" />
              </Link>
            )}
          </div>
          {job.escrow_funded && (
            <div className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 px-3 py-1 text-xs font-medium">
              <ShieldCheck className="w-3.5 h-3.5" />
              Escrow funded — payment is secured for this gig
            </div>
          )}
        </div>

        {/* CTA */}
        <div>
          {isOwner ? (
            job.status === "pending_funding" ? (
              <Button className="w-full" onClick={() => fundMutation.mutate()} disabled={fundMutation.isPending}>
                {fundMutation.isPending ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  `Fund escrow (${formatMoney(job.budget)}) to publish`
                )}
              </Button>
            ) : (
              <p className="text-sm text-muted-foreground">This is your gig posting.</p>
            )
          ) : isMusician ? (
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
          ) : null}
        </div>

        {/* Info cards */}
        <div className="grid grid-cols-2 gap-3">
          <InfoCard icon={Banknote} label="Budget (fixed price)" value={formatMoney(job.budget)} />
          <InfoCard icon={Clock} label="Duration" value={JOB_DURATION_LABELS[job.duration]} />
          <InfoCard icon={BarChart3} label="Experience level" value={JOB_EXPERIENCE_LABELS[job.experience_level]} />
          <InfoCard icon={Layers} label="Project type" value={JOB_PROJECT_TYPE_LABELS[job.project_type]} />
        </div>

        {/* Summary */}
        <div>
          <h3 className="font-semibold text-foreground mb-2">Summary</h3>
          <p className="text-sm text-foreground whitespace-pre-line leading-relaxed">{job.description}</p>
        </div>

        {/* Skills */}
        {skills.length > 0 && (
          <div>
            <h3 className="font-semibold text-foreground mb-2">Skills and experience</h3>
            <div className="flex flex-wrap gap-1.5">
              {skills.map((s) => (
                <span key={s} className="text-xs font-medium bg-muted text-foreground rounded-full px-2.5 py-1">
                  {s}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Activity */}
        <div>
          <h3 className="font-semibold text-foreground mb-2">Activity on this gig</h3>
          <p className="text-sm text-muted-foreground">
            {job.application_count === 1 ? "1 proposal submitted" : `${job.application_count || 0} proposals submitted`}
          </p>
        </div>

        {/* About the client */}
        {job.client && (
          <div className="pt-4 border-t border-border">
            <h3 className="font-semibold text-foreground mb-3 flex items-center gap-1.5">
              <ShieldCheck className="w-4 h-4 text-primary" />
              About the client
            </h3>
            <div className="space-y-2 text-sm">
              <p className="font-medium text-foreground">{job.client.company_name || job.client.name}</p>
              {job.client.location && (
                <p className="text-muted-foreground flex items-center gap-1.5">
                  <MapPin className="w-3.5 h-3.5" />
                  {job.client.location}
                </p>
              )}
              {job.client.member_since && (
                <p className="text-muted-foreground flex items-center gap-1.5">
                  <CalendarDays className="w-3.5 h-3.5" />
                  Member since {new Date(job.client.member_since).toLocaleDateString(undefined, { year: "numeric", month: "long" })}
                </p>
              )}
              <p className="text-muted-foreground flex items-center gap-1.5">
                <Star className="w-3.5 h-3.5 text-amber-500 fill-amber-500" />
                {job.client.review_count > 0 ? job.client.rating.toFixed(1) : "New"}
                <span>· {job.client.review_count || 0} reviews</span>
              </p>
            </div>

            <div className="grid grid-cols-2 gap-3 mt-4">
              <InfoCard icon={Briefcase} label="Jobs posted" value={job.client.jobs_posted} />
              <InfoCard icon={Layers} label="Open jobs" value={job.client.open_jobs} />
              {job.client.jobs_posted > 0 && (
                <InfoCard icon={TrendingUp} label="Hire rate" value={`${job.client.hire_rate.toFixed(0)}%`} />
              )}
              {job.client.total_spent > 0 && <InfoCard icon={Banknote} label="Total spent" value={formatMoney(job.client.total_spent)} />}
            </div>

            {job.client.recent_hires?.length > 0 && (
              <div className="mt-4">
                <p className="text-xs font-medium text-muted-foreground mb-2">Recently hired talent</p>
                <div className="space-y-2">
                  {job.client.recent_hires.map((h, i) => (
                    <div key={i} className="flex items-center justify-between gap-2 text-sm rounded-lg bg-muted/30 px-3 py-2">
                      <div className="min-w-0">
                        <p className="text-foreground font-medium truncate">{h.musician_name}</p>
                        <p className="text-xs text-muted-foreground truncate">{h.job_title}</p>
                      </div>
                      <span className="text-xs text-muted-foreground shrink-0 capitalize">{h.status}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        <CopyLinkButton jobId={job.id} />
      </div>
    </div>
  );
}
