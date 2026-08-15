"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiDelete } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import AdvancedFilters from "@/components/admin/AdvancedFilters";
import { formatMoney } from "@/lib/utils";
import {
  Loader2,
  MapPin,
  Briefcase,
  Trash2,
  Search,
  ChevronRight,
} from "lucide-react";
import { toast } from "sonner";

const STATUS_COLOR = {
  pending_funding: "bg-amber-500",
  open: "bg-sky-500",
  pending_hire_funding: "bg-amber-500",
  active: "bg-primary",
  completed: "bg-emerald-500",
  disputed: "bg-rose-500",
  closed: "bg-muted-foreground",
};

const emptyAdvanced = { minBudget: "", maxBudget: "", from: "", to: "" };

export default function AdminJobs() {
  const queryClient = useQueryClient();
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

  async function handleDelete(jobId) {
    try {
      await apiDelete("/admin/jobs", { job_id: jobId });
      toast.success("Job deleted.");
      queryClient.invalidateQueries({ queryKey: ["admin-jobs"] });
    } catch (err) {
      toast.error(err.message);
    }
  }

  if (isLoading) {
    return (
      <div className="flex justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-xs">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search title or location"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-44">
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
        <span className="text-sm text-muted-foreground ml-auto shrink-0">
          {filtered.length} of {jobs?.length ?? 0}
        </span>
      </div>

      {filtered.length ? (
        filtered.map((job) => (
          <Link
            key={job.id}
            href={`/admin/jobs/${job.id}`}
            className="group bg-card rounded-xl border border-border p-4 flex items-center justify-between gap-4 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30"
          >
            <div className="flex items-center gap-3 min-w-0 flex-1">
              <IconBadge
                icon={Briefcase}
                color={STATUS_COLOR[job.status] || "bg-muted-foreground"}
                size="sm"
              />
              <div className="min-w-0 flex-1">
                <p
                  className="font-medium text-foreground truncate"
                  title={job.title}
                >
                  {job.title}
                </p>
                <div className="flex items-center gap-2 flex-wrap mt-1">
                  <StatusBadge status={job.status} />
                  <span className="text-sm text-muted-foreground flex items-center gap-1.5">
                    <MapPin className="w-3.5 h-3.5 shrink-0" />
                    {job.location} · {formatMoney(job.budget)}
                  </span>
                </div>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Button
                size="sm"
                variant="destructive"
                onClick={(e) => {
                  e.preventDefault();
                  handleDelete(job.id);
                }}
                className="gap-1.5"
              >
                <Trash2 className="w-3.5 h-3.5" />
                Delete
              </Button>
              <ChevronRight className="w-4 h-4 text-muted-foreground group-hover:text-foreground transition-colors" />
            </div>
          </Link>
        ))
      ) : (
        <p className="text-center text-sm text-muted-foreground py-24">
          {jobs?.length ? "No jobs match your filters." : "No jobs posted yet."}
        </p>
      )}
    </div>
  );
}
