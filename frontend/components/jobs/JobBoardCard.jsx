"use client";

import Link from "next/link";
import IconBadge from "@/components/ui/icon-badge";
import StatusBadge from "@/components/ui/status-badge";
import SaveJobButton from "@/components/jobs/SaveJobButton";
import { Button } from "@/components/ui/button";
import { formatMoney, postedAgo } from "@/lib/utils";
import { Guitar, MapPin, Clock, ShieldCheck, Star, Users } from "lucide-react";

// Shared job row used across every tab of the talent job board (Best
// matches, Most recent, Saved jobs, Invites-adjacent). Saved jobs aren't
// filtered by status server-side, so a job that's since been filled shows
// a "Closed" badge here instead of silently looking identical to an open one.
//
// The whole card (including the "Apply now" button) opens the Upwork-style
// detail panel via the parent's onOpen, which sets a shareable ?job= URL
// param — applying happens from inside that panel, not straight off the
// card. It's still a real <Link>, so ctrl/cmd/middle-click still opens the
// full job page in a new tab as expected.
export default function JobBoardCard({ job, saved, alreadyApplied, onOpen, showSaveButton = true }) {
  const isClosed = job.status !== "open";

  function handleClick(e) {
    if (!onOpen) return;
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button === 1) return;
    e.preventDefault();
    onOpen(job.id);
  }

  return (
    <Link
      href={`/jobs/${job.id}`}
      onClick={handleClick}
      className="group block bg-card rounded-2xl border border-border p-5 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30"
    >
      <div className="flex items-start gap-4">
        <IconBadge icon={Guitar} color={isClosed ? "bg-muted-foreground" : "bg-primary"} />
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-3">
            <h3 className="font-semibold text-foreground truncate group-hover:text-primary transition-colors min-w-0" title={job.title}>
              {job.title}
            </h3>
            <div className="flex items-center gap-1 shrink-0">
              {isClosed && <StatusBadge status="closed" label="Closed" />}
              {showSaveButton && <SaveJobButton jobId={job.id} saved={saved} />}
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 mt-0.5 text-xs text-muted-foreground">
            {postedAgo(job.created_at) && (
              <span className="flex items-center gap-1">
                <Clock className="w-3 h-3" />
                {postedAgo(job.created_at)}
              </span>
            )}
            <span className="flex items-center gap-1">
              <Users className="w-3 h-3" />
              {job.application_count === 1 ? "1 proposal" : `${job.application_count || 0} proposals`}
            </span>
          </div>

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

          <div className="flex items-center justify-between gap-3 mt-3 pt-3 border-t border-border/60">
            <div className="flex items-center gap-3 text-xs text-muted-foreground">
              {job.escrow_funded && (
                <span className="flex items-center gap-1 text-emerald-600 dark:text-emerald-400 font-medium">
                  <ShieldCheck className="w-3.5 h-3.5" />
                  Escrow funded
                </span>
              )}
              {job.client_review_count > 0 && (
                <span className="flex items-center gap-1">
                  <Star className="w-3.5 h-3.5 text-amber-500 fill-amber-500" />
                  {job.client_rating.toFixed(1)}
                </span>
              )}
            </div>
            {isClosed ? (
              <Button disabled variant="outline" size="sm">
                No longer available
              </Button>
            ) : alreadyApplied ? (
              <Button disabled variant="outline" size="sm">
                Applied
              </Button>
            ) : (
              <Button size="sm" onClick={handleClick}>
                Apply now
              </Button>
            )}
          </div>
        </div>
      </div>
    </Link>
  );
}
