"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import IconBadge from "@/components/ui/icon-badge";
import SaveJobButton from "@/components/jobs/SaveJobButton";
import JobApplyModal from "@/components/jobs/JobApplyModal";
import { instrumentIcon } from "@/lib/instrument-icons";
import { cn, formatMoney, postedAgo, JOB_DURATION_LABELS, JOB_EXPERIENCE_LABELS, JOB_PROJECT_TYPE_LABELS } from "@/lib/utils";
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

// A vertical label/value list rather than a fixed grid of tiles — any subset
// of rows can be present (budget always is, duration/experience/project type
// are all optional) without ever leaving a lopsided half-empty grid row.
function SpecList({ items }) {
  const rows = items.filter((row) => row.value);
  if (!rows.length) return null;
  return (
    <div>
      {rows.map((row) => (
        <div key={row.label} className="flex items-center justify-between gap-3 py-2.5 border-b border-border/60 last:border-0">
          <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <row.icon className="w-3.5 h-3.5" />
            {row.label}
          </span>
          <span className="text-sm font-semibold text-foreground text-right">{row.value}</span>
        </div>
      ))}
    </div>
  );
}

function CopyLinkButton({ jobId }) {
  const [copied, setCopied] = useState(false);
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      className="gap-1.5 text-muted-foreground hover:text-foreground -ml-3"
      onClick={() => {
        const url = `${window.location.origin}/jobs/${jobId}`;
        navigator.clipboard.writeText(url).then(() => {
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        });
      }}
    >
      {copied ? <Check className="w-3.5 h-3.5 text-status-success" /> : <Link2 className="w-3.5 h-3.5" />}
      {copied ? "Link copied" : "Copy job link"}
    </Button>
  );
}

function ClientCard({ client }) {
  return (
    <Card>
      <div className="px-(--card-spacing)">
        <h3 className="font-semibold text-foreground mb-3 flex items-center gap-1.5">
          <ShieldCheck className="w-4 h-4 text-primary" />
          About the client
        </h3>
        <div className="space-y-2 text-sm">
          <p className="font-medium text-foreground">{client.company_name || client.name}</p>
          {client.location && (
            <p className="text-muted-foreground flex items-center gap-1.5">
              <MapPin className="w-3.5 h-3.5" />
              {client.location}
            </p>
          )}
          {client.member_since && (
            <p className="text-muted-foreground flex items-center gap-1.5">
              <CalendarDays className="w-3.5 h-3.5" />
              Member since {new Date(client.member_since).toLocaleDateString(undefined, { year: "numeric", month: "long" })}
            </p>
          )}
          <p className="text-muted-foreground flex items-center gap-1.5">
            <Star className="w-3.5 h-3.5 text-status-warning fill-status-warning" />
            {client.review_count > 0 ? client.rating.toFixed(1) : "New"}
            <span>· {client.review_count || 0} reviews</span>
          </p>
        </div>

        <div className="mt-3">
          <SpecList
            items={[
              { icon: Briefcase, label: "Jobs posted", value: client.jobs_posted },
              { icon: Layers, label: "Open jobs", value: client.open_jobs },
              { icon: TrendingUp, label: "Hire rate", value: client.jobs_posted > 0 ? `${client.hire_rate.toFixed(0)}%` : null },
              { icon: Banknote, label: "Total spent", value: client.total_spent > 0 ? formatMoney(client.total_spent) : null },
            ]}
          />
        </div>

        {client.recent_hires?.length > 0 && (
          <div className="mt-4">
            <p className="text-xs font-medium text-muted-foreground mb-2">Recently hired talent</p>
            <div className="space-y-2">
              {client.recent_hires.map((h, i) => (
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
    </Card>
  );
}

// Renders in two contexts: a narrow slide-out Sheet from the job board
// (layout="stacked", the default) and the full standalone /jobs/[id] page
// (layout="split", a wider two-column treatment with a sticky sidebar).
export default function JobDetailContent({
  job,
  currentUser,
  isAuthenticated,
  alreadyApplied,
  saved,
  onApplied,
  showOpenInNewWindow = false,
  layout = "stacked",
}) {
  const queryClient = useQueryClient();
  const isMusician = currentUser?.role === "musician";
  const isOwner = currentUser?.id === job.client_id;
  const isClosed = job.status !== "open";
  const skills = job.skills?.length ? job.skills : [job.instrument, job.genre].filter(Boolean);
  const isSplit = layout === "split";

  const goLiveMutation = useMutation({
    mutationFn: () => apiPost("/jobs/fund", { job_id: job.id }),
    onSuccess: () => {
      toast.success("Your gig is now live and open to applicants.");
      queryClient.invalidateQueries({ queryKey: ["job", job.id] });
      queryClient.invalidateQueries({ queryKey: ["client-jobs"] });
    },
    onError: (err) => toast.error(err.message),
  });

  const header = (
    <div>
      <div className="flex items-start gap-4">
        <IconBadge icon={instrumentIcon(job.instrument)} color={isClosed ? "bg-muted-foreground" : "bg-primary"} size={isSplit ? "lg" : "md"} />
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-3">
            <h2 className={cn("font-bold text-foreground leading-snug", isSplit ? "text-2xl sm:text-3xl" : "text-lg")}>{job.title}</h2>
            {isMusician && !isClosed && <SaveJobButton jobId={job.id} saved={saved} className="shrink-0" />}
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
              <Link href={`/jobs/${job.id}`} target="_blank" className="flex items-center gap-1 text-primary hover:underline">
                Open in a new window
                <ExternalLink className="w-3 h-3" />
              </Link>
            )}
          </div>
        </div>
      </div>
      {job.escrow_funded && (
        <div className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-status-success/10 text-status-success px-3 py-1 text-xs font-medium">
          <ShieldCheck className="w-3.5 h-3.5" />
          Escrow funded — payment is secured for this gig
        </div>
      )}
    </div>
  );

  const applyCta = isOwner ? (
    job.status === "pending_funding" ? (
      <Button className="w-full" onClick={() => goLiveMutation.mutate()} disabled={goLiveMutation.isPending}>
        {goLiveMutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : "Go live"}
      </Button>
    ) : (
      <p className="text-sm text-muted-foreground text-center">This is your gig posting.</p>
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
    <Link href="/login" className="block">
      <Button className="w-full">Log in to apply</Button>
    </Link>
  ) : null;

  const specs = (
    <SpecList
      items={[
        { icon: Banknote, label: "Budget (fixed price)", value: formatMoney(job.budget) },
        { icon: Clock, label: "Duration", value: JOB_DURATION_LABELS[job.duration] },
        { icon: BarChart3, label: "Experience level", value: JOB_EXPERIENCE_LABELS[job.experience_level] },
        { icon: Layers, label: "Project type", value: JOB_PROJECT_TYPE_LABELS[job.project_type] },
      ]}
    />
  );

  const body = (
    <>
      <div>
        <h3 className="font-semibold text-foreground mb-2">Summary</h3>
        <p className="text-sm text-foreground whitespace-pre-line leading-relaxed">{job.description}</p>
      </div>

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

      <div>
        <h3 className="font-semibold text-foreground mb-2">Activity on this gig</h3>
        <p className="text-sm text-muted-foreground">
          {job.application_count === 1 ? "1 proposal submitted" : `${job.application_count || 0} proposals submitted`}
        </p>
      </div>
    </>
  );

  if (isSplit) {
    return (
      <div className="grid lg:grid-cols-3 gap-8 items-start">
        <div className="lg:col-span-2 space-y-8">
          {header}
          {body}
        </div>
        <div className="space-y-6 lg:sticky lg:top-24">
          <Card>
            <div className="px-(--card-spacing) space-y-4">
              {specs}
              {applyCta}
            </div>
          </Card>
          {job.client && <ClientCard client={job.client} />}
          <CopyLinkButton jobId={job.id} />
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col overflow-y-auto">
      <div className="p-4 sm:p-5 space-y-6">
        {header}
        {specs}
        {/* Apply sits right under the specs, same as the split layout's
            sidebar — not after "About the client", which can run long
            enough that the action was easy to miss without scrolling past
            it entirely. */}
        {applyCta}
        {body}
        {job.client && <ClientCard client={job.client} />}
        <CopyLinkButton jobId={job.id} />
      </div>
    </div>
  );
}
