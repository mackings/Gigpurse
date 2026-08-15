"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import StatusBadge from "@/components/ui/status-badge";
import { formatMoney } from "@/lib/utils";
import {
  ArrowLeft,
  Briefcase,
  Handshake,
  Users,
  MapPin,
  Loader2,
  Star,
} from "lucide-react";

function Section({ title, icon: Icon, children }) {
  return (
    <div className="bg-card rounded-2xl border border-border p-5">
      <h2 className="font-semibold text-foreground flex items-center gap-2 mb-4">
        <Icon className="w-4 h-4 text-muted-foreground" />
        {title}
      </h2>
      {children}
    </div>
  );
}

export default function AdminJobDetail() {
  const { id } = useParams();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-job-detail", id],
    queryFn: () => apiGet(`/admin/jobs/${id}`),
  });

  if (isLoading) {
    return (
      <div className="flex justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }
  if (!data?.job) {
    return (
      <p className="text-center text-sm text-muted-foreground py-24">
        Job not found.
      </p>
    );
  }

  const { job, applications, contract, milestones } = data;

  return (
    <div className="space-y-4">
      <Link
        href="/admin/jobs"
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="w-4 h-4" />
        Back to jobs
      </Link>

      <Section title="Job" icon={Briefcase}>
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div>
            <div className="flex items-center gap-2 flex-wrap">
              <h1 className="text-lg font-semibold text-foreground">
                {job.title}
              </h1>
              <StatusBadge status={job.status} />
            </div>
            <p className="text-sm text-muted-foreground mt-1 flex items-center gap-1.5">
              <MapPin className="w-3.5 h-3.5" />
              {job.location} · {job.instrument} · {job.genre}
            </p>
            <p className="text-sm text-muted-foreground mt-2 max-w-xl">
              {job.description}
            </p>
          </div>
          <div className="text-right shrink-0">
            <p className="text-2xl font-bold text-foreground tabular-nums">
              {formatMoney(job.budget)}
            </p>
            <p className="text-xs text-muted-foreground">
              Posted {new Date(job.created_at).toLocaleDateString()}
            </p>
          </div>
        </div>
      </Section>

      <Section
        title={`Applications (${applications?.length ?? 0})`}
        icon={Users}
      >
        {applications?.length ? (
          <div className="space-y-3">
            {applications.map((app) => (
              <div
                key={app.id}
                className="flex items-center justify-between gap-4 p-3 rounded-lg bg-muted/40"
              >
                <div className="min-w-0">
                  <p className="font-medium text-foreground truncate">
                    {app.applicant?.name || "Unknown applicant"}
                  </p>
                  <p className="text-sm text-muted-foreground truncate">
                    {app.proposal}
                  </p>
                  {app.applicant?.rating > 0 && (
                    <p className="text-xs text-muted-foreground flex items-center gap-1 mt-0.5">
                      <Star className="w-3 h-3 fill-amber-500 text-amber-500" />
                      {app.applicant.rating.toFixed(1)} (
                      {app.applicant.review_count})
                    </p>
                  )}
                </div>
                <div className="text-right shrink-0">
                  <p className="font-semibold text-foreground tabular-nums">
                    {formatMoney(app.price_bid)}
                  </p>
                  <StatusBadge status={app.status} />
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">No applications yet.</p>
        )}
      </Section>

      {contract && (
        <Section title="Contract & escrow" icon={Handshake}>
          <div className="flex items-center justify-between gap-4 pb-4 mb-4 border-b border-border">
            <div>
              <p className="font-medium text-foreground">{contract.title}</p>
              <p className="text-xs text-muted-foreground">
                Created {new Date(contract.created_at).toLocaleDateString()}
              </p>
            </div>
            <StatusBadge status={contract.status} />
          </div>
          {milestones?.length ? (
            <div className="space-y-3">
              {milestones.map((m) => (
                <div
                  key={m.id}
                  className="flex items-center justify-between gap-4 p-3 rounded-lg bg-muted/40"
                >
                  <div className="min-w-0">
                    <p className="font-medium text-foreground truncate">
                      {m.title}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {m.escrow_status
                        ? `Escrow: ${m.escrow_status.toLowerCase()}`
                        : "Not yet funded"}
                      {m.payout_status &&
                        m.payout_status !== "NONE" &&
                        ` · Payout: ${m.payout_status.toLowerCase()}`}
                      {m.refund_status &&
                        m.refund_status !== "NONE" &&
                        ` · Refund: ${m.refund_status.toLowerCase()}`}
                    </p>
                  </div>
                  <div className="text-right shrink-0">
                    <p className="font-semibold text-foreground tabular-nums">
                      {formatMoney(m.amount)}
                    </p>
                    <StatusBadge status={m.status} />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              No milestones proposed yet.
            </p>
          )}
        </Section>
      )}
    </div>
  );
}
