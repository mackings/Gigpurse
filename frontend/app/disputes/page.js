"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { Loader2, ShieldAlert } from "lucide-react";

const STATUS_COLOR = { open: "bg-rose-500", resolved: "bg-emerald-500", closed: "bg-muted-foreground" };

export default function DisputesPage() {
  const { data: disputes, isLoading } = useQuery({
    queryKey: ["disputes", "mine"],
    queryFn: () => apiGet("/disputes"),
  });

  return (
    <div className="min-h-screen bg-background py-12 px-4">
      <div className="max-w-3xl mx-auto space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-foreground tracking-tight">Disputes</h1>
          <p className="text-muted-foreground">
            Track disputes you&apos;ve opened. Open a new one from the relevant contract&apos;s page.
          </p>
        </div>

        {isLoading ? (
          <div className="flex justify-center py-24">
            <Loader2 className="w-8 h-8 animate-spin text-primary" />
          </div>
        ) : disputes?.length ? (
          <div className="space-y-3">
            {disputes.map((d) => (
              <Link
                key={d.id}
                href={`/contracts/${d.contract_id}`}
                className="group block bg-card rounded-xl border border-border p-4 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-rose-500/30 hover:-translate-y-0.5"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-start gap-3 min-w-0">
                    <IconBadge icon={ShieldAlert} color={STATUS_COLOR[d.status] || "bg-muted-foreground"} size="sm" />
                    <div className="min-w-0">
                      <p className="font-medium text-foreground">{d.reason}</p>
                      <p className="text-sm text-muted-foreground mt-1">
                        Opened {new Date(d.created_at).toLocaleDateString()}
                      </p>
                      {d.resolution && <p className="text-sm text-muted-foreground mt-1">Resolution: {d.resolution}</p>}
                    </div>
                  </div>
                  <StatusBadge status={d.status} />
                </div>
              </Link>
            ))}
          </div>
        ) : (
          <div className="text-center py-24 text-muted-foreground">
            <ShieldAlert className="w-8 h-8 mx-auto mb-2" />
            <p className="text-sm">No disputes yet.</p>
          </div>
        )}
      </div>
    </div>
  );
}
