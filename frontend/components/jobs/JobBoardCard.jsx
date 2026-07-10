"use client";

import Link from "next/link";
import IconBadge from "@/components/ui/icon-badge";
import StatusBadge from "@/components/ui/status-badge";
import SaveJobButton from "@/components/jobs/SaveJobButton";
import JobApplyModal from "@/components/jobs/JobApplyModal";
import { Button } from "@/components/ui/button";
import { formatMoney } from "@/lib/utils";
import { Guitar, MapPin, Clock } from "lucide-react";

function postedAgo(dateStr) {
  if (!dateStr) return null;
  const diffMs = Date.now() - new Date(dateStr).getTime();
  if (!Number.isFinite(diffMs) || diffMs < 0) return null;
  const days = Math.floor(diffMs / 86400000);
  if (days < 1) return "Posted today";
  if (days === 1) return "Posted yesterday";
  if (days < 30) return `Posted ${days} days ago`;
  return `Posted ${new Date(dateStr).toLocaleDateString()}`;
}

// Shared job row used across every tab of the talent job board (Best
// matches, Most recent, Saved jobs, Invites-adjacent). Saved jobs aren't
// filtered by status server-side, so a job that's since been filled shows
// a "Closed" badge here instead of silently looking identical to an open one.
export default function JobBoardCard({ job, saved, alreadyApplied, onApplied, showSaveButton = true }) {
  const isClosed = job.status !== "open";

  return (
    <div className="group bg-card rounded-2xl border border-border p-5 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30">
      <div className="flex items-start gap-4">
        <IconBadge icon={Guitar} color={isClosed ? "bg-muted-foreground" : "bg-primary"} />
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-3">
            <Link href={`/jobs/${job.id}`} className="min-w-0">
              <h3 className="font-semibold text-foreground truncate hover:text-primary transition-colors" title={job.title}>
                {job.title}
              </h3>
            </Link>
            <div className="flex items-center gap-1 shrink-0">
              {isClosed && <StatusBadge status="closed" label="Closed" />}
              {showSaveButton && <SaveJobButton jobId={job.id} saved={saved} />}
            </div>
          </div>

          {postedAgo(job.created_at) && (
            <p className="text-xs text-muted-foreground flex items-center gap-1 mt-0.5">
              <Clock className="w-3 h-3" />
              {postedAgo(job.created_at)}
            </p>
          )}

          <p className="text-muted-foreground mt-2 line-clamp-2">{job.description}</p>

          <div className="flex flex-wrap items-center gap-3 mt-3 text-sm text-muted-foreground">
            {job.location && (
              <span className="flex items-center gap-1">
                <MapPin className="w-3.5 h-3.5" />
                {job.location}
              </span>
            )}
            <span className="font-medium text-foreground">{formatMoney(job.budget)}</span>
            {job.instrument && <span>{job.instrument}</span>}
            {job.genre && <span>{job.genre}</span>}
          </div>

          <div className="mt-4">
            {isClosed ? (
              <Button disabled variant="outline" size="sm">
                No longer available
              </Button>
            ) : alreadyApplied ? (
              <Button disabled variant="outline" size="sm">
                Applied
              </Button>
            ) : (
              <JobApplyModal job={job} trigger={<Button size="sm">Apply now</Button>} onApplied={onApplied} />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
