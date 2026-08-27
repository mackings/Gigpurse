"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import StatusBadge from "@/components/ui/status-badge";
import { formatMoney, postedAgo, initials } from "@/lib/utils";
import {
  ArrowLeft,
  Briefcase,
  Handshake,
  Users,
  MapPin,
  Loader2,
} from "lucide-react";

function Section({ title, icon: Icon, children }) {
  return (
    <div className="bg-card rounded-xl border border-border p-6">
      <h2 className="font-semibold text-foreground flex items-center gap-2 mb-4">
        <Icon className="w-4 h-4 text-muted-foreground" />
        {title}
      </h2>
      {children}
    </div>
  );
}

function PersonRow({ name, subtitle, amount, status }) {
  return (
    <div className="flex items-center justify-between gap-4 p-3.5 rounded-lg border border-border">
      <div className="flex items-center gap-3 min-w-0">
        <div className="w-9 h-9 rounded-full bg-muted flex items-center justify-center shrink-0 text-xs font-semibold text-foreground">
          {initials(name)}
        </div>
        <div className="min-w-0">
          <p className="font-medium text-foreground truncate">{name}</p>
          {subtitle && (
            <p className="text-sm text-muted-foreground truncate">{subtitle}</p>
          )}
        </div>
      </div>
      <div className="text-right shrink-0">
        <p className="font-semibold text-foreground tabular-nums">{amount}</p>
        {status}
      </div>
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
        className="lg:hidden inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="w-4 h-4" />
        Back to jobs
      </Link>

      <div className="bg-card rounded-xl border border-border p-6">
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div className="flex items-start gap-4 min-w-0">
            <div className="w-12 h-12 rounded-lg bg-primary flex items-center justify-center shrink-0">
              <Briefcase className="w-6 h-6 text-primary-foreground" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <h1 className="text-lg font-semibold text-foreground">
                  {job.title}
                </h1>
                <StatusBadge status={job.status} />
              </div>
              <p className="text-sm text-muted-foreground mt-1 flex items-center gap-1.5">
                <MapPin className="w-3.5 h-3.5 shrink-0" />
                {job.location} · {job.instrument} · {job.genre}
              </p>
              <p className="text-sm text-muted-foreground mt-2 max-w-xl">
                {job.description}
              </p>
            </div>
          </div>
          <div className="text-right shrink-0">
            <p className="text-3xl font-bold text-foreground tabular-nums tracking-tight">
              {formatMoney(job.budget)}
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              {postedAgo(job.created_at)}
            </p>
          </div>
        </div>
      </div>

      <Section
        title={`Applications (${applications?.length ?? 0})`}
        icon={Users}
      >
        {applications?.length ? (
          <div className="space-y-2.5">
            {applications.map((app) => (
              <PersonRow
                key={app.id}
                name={app.applicant?.name || "Unknown applicant"}
                subtitle={
                  app.applicant?.rating > 0
                    ? `★ ${app.applicant.rating.toFixed(1)} (${app.applicant.review_count}) · ${app.proposal}`
                    : app.proposal
                }
                amount={formatMoney(app.price_bid)}
                status={<StatusBadge status={app.status} />}
              />
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
            <div className="space-y-2.5">
              {milestones.map((m) => (
                <PersonRow
                  key={m.id}
                  name={m.title}
                  subtitle={[
                    m.escrow_status
                      ? `Escrow: ${m.escrow_status.toLowerCase()}`
                      : "Not yet funded",
                    m.payout_status && m.payout_status !== "NONE" && `Payout: ${m.payout_status.toLowerCase()}`,
                    m.refund_status && m.refund_status !== "NONE" && `Refund: ${m.refund_status.toLowerCase()}`,
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                  amount={formatMoney(m.amount)}
                  status={<StatusBadge status={m.status} />}
                />
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
