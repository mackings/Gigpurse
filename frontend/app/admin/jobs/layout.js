"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import StatusBadge from "@/components/ui/status-badge";
import AdvancedFilters from "@/components/admin/AdvancedFilters";
import { cn, formatMoney } from "@/lib/utils";
import { Loader2, Briefcase, Search, MapPin, Users } from "lucide-react";

// Mirrors components/ui/status-badge.jsx's semantic mapping (same
// --status-* tokens) rather than its own hand-picked Tailwind literals.
const STATUS_COLOR = {
  pending_funding: "bg-status-warning",
  open: "bg-status-info",
  pending_hire_funding: "bg-status-warning",
  active: "bg-primary",
  completed: "bg-status-success",
  disputed: "bg-status-critical",
  closed: "bg-muted-foreground",
};

const emptyAdvanced = { minBudget: "", maxBudget: "", from: "", to: "" };

// Master-detail shell shared by /admin/jobs and /admin/jobs/[id] — the list
// (with search/filters) stays mounted across navigation between jobs
// instead of refetching, and {children} renders either the "pick a job"
// empty state or the selected job's detail in the right-hand pane. Below
// the lg breakpoint there's no room for both columns side by side, so only
// one shows at a time based on whether a specific job is selected.
export default function AdminJobsLayout({ children }) {
  const pathname = usePathname();
  const isDetailView = pathname !== "/admin/jobs";

  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [advanced, setAdvanced] = useState(emptyAdvanced);
  const { data: jobs, isLoading } = useQuery({
    queryKey: ["admin-jobs"],
    queryFn: () => apiGet("/admin/jobs"),
  });

  const activeAdvancedCount = Object.values(advanced).filter(Boolean).length;

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return (jobs || []).filter((j) => {
      if (status !== "all" && j.status !== status) return false;
      if (
        q &&
        !j.title?.toLowerCase().includes(q) &&
        !j.location?.toLowerCase().includes(q)
      )
        return false;
      if (advanced.minBudget && j.budget < Number(advanced.minBudget))
        return false;
      if (advanced.maxBudget && j.budget > Number(advanced.maxBudget))
        return false;
      if (advanced.from && new Date(j.created_at) < new Date(advanced.from))
        return false;
      if (
        advanced.to &&
        new Date(j.created_at) > new Date(advanced.to + "T23:59:59")
      )
        return false;
      return true;
    });
  }, [jobs, search, status, advanced]);

  return (
    <div className="lg:grid lg:grid-cols-[440px_1fr] lg:gap-6 lg:items-start">
      <div className={cn("space-y-3", isDetailView && "hidden lg:block")}>
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search title or location"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
          <AdvancedFilters
            activeCount={activeAdvancedCount}
            onClear={() => setAdvanced(emptyAdvanced)}
          >
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                Budget range (₦)
              </label>
              <div className="flex items-center gap-2">
                <Input
                  type="number"
                  placeholder="Min"
                  value={advanced.minBudget}
                  onChange={(e) =>
                    setAdvanced({ ...advanced, minBudget: e.target.value })
                  }
                />
                <Input
                  type="number"
                  placeholder="Max"
                  value={advanced.maxBudget}
                  onChange={(e) =>
                    setAdvanced({ ...advanced, maxBudget: e.target.value })
                  }
                />
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                Posted between
              </label>
              <div className="flex items-center gap-2">
                <Input
                  type="date"
                  value={advanced.from}
                  onChange={(e) =>
                    setAdvanced({ ...advanced, from: e.target.value })
                  }
                />
                <Input
                  type="date"
                  value={advanced.to}
                  onChange={(e) =>
                    setAdvanced({ ...advanced, to: e.target.value })
                  }
                />
              </div>
            </div>
          </AdvancedFilters>
        </div>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            <SelectItem value="pending_funding">Draft</SelectItem>
            <SelectItem value="open">Open</SelectItem>
            <SelectItem value="pending_hire_funding">Pending hire</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="completed">Completed</SelectItem>
            <SelectItem value="disputed">Disputed</SelectItem>
            <SelectItem value="closed">Closed</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground">
          {filtered.length} of {jobs?.length ?? 0}
        </p>

        {isLoading ? (
          <div className="flex justify-center py-24">
            <Loader2 className="w-6 h-6 animate-spin text-primary" />
          </div>
        ) : filtered.length ? (
          <div className="space-y-2">
            {filtered.map((job) => {
              const active = pathname === `/admin/jobs/${job.id}`;
              return (
                <Link
                  key={job.id}
                  href={`/admin/jobs/${job.id}`}
                  className={cn(
                    "block rounded-lg border p-3 transition-colors",
                    active
                      ? "bg-accent border-primary/30"
                      : "bg-card border-border hover:border-foreground/15",
                  )}
                >
                  <div className="flex items-start gap-3">
                    <div
                      className={`w-9 h-9 rounded-md flex items-center justify-center shrink-0 ${STATUS_COLOR[job.status] || "bg-muted-foreground"}`}
                    >
                      <Briefcase className="w-4.5 h-4.5 text-white" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium text-foreground truncate">
                        {job.title}
                      </p>
                      {job.client?.name && (
                        <p className="text-xs text-muted-foreground truncate mt-0.5">
                          by {job.client.company_name || job.client.name}
                        </p>
                      )}
                      <p className="text-sm font-semibold text-foreground tabular-nums mt-1.5">
                        {formatMoney(job.budget)}
                      </p>
                      <div className="flex items-center gap-x-3 gap-y-1 flex-wrap text-xs text-muted-foreground mt-1.5">
                        {job.location && (
                          <span className="flex items-center gap-1">
                            <MapPin className="w-3 h-3 shrink-0" />
                            {job.location}
                          </span>
                        )}
                        {typeof job.application_count === "number" && (
                          <span className="flex items-center gap-1">
                            <Users className="w-3 h-3 shrink-0" />
                            {job.application_count} applicant{job.application_count === 1 ? "" : "s"}
                          </span>
                        )}
                      </div>
                      <div className="mt-1.5">
                        <StatusBadge status={job.status} />
                      </div>
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        ) : (
          <p className="text-center text-sm text-muted-foreground py-24">
            {jobs?.length ? "No jobs match your filters." : "No jobs posted yet."}
          </p>
        )}
      </div>

      <div className={cn(!isDetailView && "hidden lg:block")}>{children}</div>
    </div>
  );
}
