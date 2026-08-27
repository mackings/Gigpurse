"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { instrumentIcon } from "@/lib/instrument-icons";
import { formatMoney } from "@/lib/utils";
import { Loader2, MapPin, Banknote } from "lucide-react";

const STATUS_COLOR = {
  pending: "bg-status-warning",
  active: "bg-primary",
  completed: "bg-status-success",
};

const tabs = [
  { value: "pending", label: "Pending" },
  { value: "active", label: "Active" },
  { value: "completed", label: "Completed" },
];

export default function MyJobs() {
  const [status, setStatus] = useState("pending");

  const { data: jobs, isLoading } = useQuery({
    queryKey: ["jobs", "mine", status],
    queryFn: () => apiGet(`/jobs/mine?status=${status}`),
  });

  return (
    <div>
      <Tabs value={status} onValueChange={setStatus} className="mb-6">
        <TabsList>
          {tabs.map((t) => (
            <TabsTrigger key={t.value} value={t.value}>
              {t.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      {isLoading ? (
        <div className="flex justify-center py-24">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
        </div>
      ) : jobs?.length ? (
        <div className="space-y-4">
          {jobs.map((job) => (
            <Card key={job.id} className="transition-colors duration-200 hover:border-foreground/15">
              <div className="px-(--card-spacing) flex items-start justify-between gap-4">
                <div className="flex items-start gap-3 min-w-0">
                  <IconBadge icon={instrumentIcon(job.instrument)} color={STATUS_COLOR[job.status] || "bg-muted-foreground"} />
                  <div className="min-w-0">
                    <h3 className="text-lg font-semibold text-foreground">{job.title}</h3>
                    <p className="text-muted-foreground mt-1 line-clamp-2">{job.description}</p>
                    <div className="flex flex-wrap gap-3 mt-3 text-sm text-muted-foreground">
                      {job.location && (
                        <span className="flex items-center gap-1">
                          <MapPin className="w-4 h-4" />
                          {job.location}
                        </span>
                      )}
                      <span className="flex items-center gap-1 font-semibold text-foreground">
                        <Banknote className="w-4 h-4" />
                        {formatMoney(job.budget)}
                      </span>
                      {job.instrument && <span>{job.instrument}</span>}
                      {job.genre && <span>{job.genre}</span>}
                    </div>
                  </div>
                </div>
                <StatusBadge status={job.status} />
              </div>
            </Card>
          ))}
        </div>
      ) : (
        <div className="text-center py-24 text-muted-foreground">No {status} jobs yet.</div>
      )}
    </div>
  );
}
