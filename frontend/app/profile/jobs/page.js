"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Loader2, MapPin, DollarSign } from "lucide-react";

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
      <div className="flex gap-2 mb-6">
        {tabs.map((t) => (
          <Button
            key={t.value}
            variant={status === t.value ? "default" : "outline"}
            size="sm"
            onClick={() => setStatus(t.value)}
          >
            {t.label}
          </Button>
        ))}
      </div>

      {isLoading ? (
        <div className="flex justify-center py-24">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
        </div>
      ) : jobs?.length ? (
        <div className="space-y-4">
          {jobs.map((job) => (
            <div key={job.id} className="bg-card rounded-2xl border border-border p-6">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h3 className="text-lg font-semibold text-foreground">{job.title}</h3>
                  <p className="text-muted-foreground mt-1">{job.description}</p>
                  <div className="flex flex-wrap gap-3 mt-3 text-sm text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <MapPin className="w-4 h-4" />
                      {job.location}
                    </span>
                    <span className="flex items-center gap-1">
                      <DollarSign className="w-4 h-4" />
                      {job.budget}
                    </span>
                    <span>{job.instrument}</span>
                    <span>{job.genre}</span>
                  </div>
                </div>
                <Badge variant="secondary" className="capitalize shrink-0">{job.status}</Badge>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="text-center py-24 text-muted-foreground">No {status} jobs yet.</div>
      )}
    </div>
  );
}
