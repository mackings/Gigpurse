"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import JobApplyModal from "@/components/jobs/JobApplyModal";
import IconBadge from "@/components/ui/icon-badge";
import { formatMoney } from "@/lib/utils";
import { Loader2, MapPin, Banknote, Sparkles, Guitar } from "lucide-react";

function buildQuery(filters) {
  const params = new URLSearchParams();
  params.set("status", "open");
  if (filters.genre) params.set("genre", filters.genre);
  if (filters.instrument) params.set("instrument", filters.instrument);
  if (filters.location) params.set("location", filters.location);
  if (filters.min_budget) params.set("min_budget", filters.min_budget);
  if (filters.max_budget) params.set("max_budget", filters.max_budget);
  params.set("sort_by", filters.sort_by || "newest");
  return params.toString();
}

function JobCard({ job, alreadyApplied, onApplied }) {
  return (
    <div className="group bg-card rounded-2xl border border-border p-6 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30 hover:-translate-y-0.5">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 min-w-0">
          <IconBadge icon={Guitar} color="bg-primary" />
          <div className="min-w-0">
            <h3 className="text-lg font-semibold text-foreground">{job.title}</h3>
            <p className="text-muted-foreground mt-1">{job.description}</p>
            <div className="flex flex-wrap gap-3 mt-3 text-sm text-muted-foreground">
              <span className="flex items-center gap-1">
                <MapPin className="w-4 h-4" />
                {job.location}
              </span>
              <span className="flex items-center gap-1">
                <Banknote className="w-4 h-4" />
                {formatMoney(job.budget)}
              </span>
              <span>{job.instrument}</span>
              <span>{job.genre}</span>
            </div>
          </div>
        </div>
        {alreadyApplied ? (
          <Button disabled variant="outline" className="shrink-0">
            Applied
          </Button>
        ) : (
          <JobApplyModal job={job} trigger={<Button className="shrink-0">Apply</Button>} onApplied={onApplied} />
        )}
      </div>
    </div>
  );
}

export default function JobBoard() {
  const { user } = useCurrentUser();
  const queryClient = useQueryClient();
  const [tab, setTab] = useState("browse");
  const [filters, setFilters] = useState({
    genre: "",
    instrument: "",
    location: "",
    min_budget: "",
    max_budget: "",
    sort_by: "newest",
  });

  const { data: jobs, isLoading } = useQuery({
    queryKey: ["jobs", "open", filters],
    queryFn: () => apiGet(`/jobs?${buildQuery(filters)}`),
    enabled: tab === "browse",
  });

  const { data: recommended, isLoading: isLoadingRecommended } = useQuery({
    queryKey: ["jobs", "recommended"],
    queryFn: () => apiGet("/jobs/recommended"),
    enabled: tab === "for-you",
  });

  const { data: myApplications } = useQuery({
    queryKey: ["applications", "mine"],
    queryFn: () => apiGet("/jobs/applications"),
  });

  const appliedJobIds = new Set((myApplications || []).map((a) => a.job_id));
  const onApplied = () => queryClient.invalidateQueries({ queryKey: ["applications", "mine"] });

  const list = tab === "for-you" ? recommended : jobs;
  const loading = tab === "for-you" ? isLoadingRecommended : isLoading;

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-4xl mx-auto px-4 py-12">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-foreground mb-2 tracking-tight">Find Gigs</h1>
          <p className="text-muted-foreground">Browse open jobs posted by clients.</p>
        </div>

        {user?.role === "musician" && (
          <div className="flex gap-2 mb-6">
            <Button variant={tab === "browse" ? "default" : "outline"} size="sm" onClick={() => setTab("browse")}>
              Browse
            </Button>
            <Button variant={tab === "for-you" ? "default" : "outline"} size="sm" onClick={() => setTab("for-you")} className="gap-1.5">
              <Sparkles className="w-3.5 h-3.5" />
              For You
            </Button>
          </div>
        )}

        {tab === "browse" && (
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-3 mb-8">
            <Input
              placeholder="Genre"
              value={filters.genre}
              onChange={(e) => setFilters({ ...filters, genre: e.target.value })}
            />
            <Input
              placeholder="Instrument"
              value={filters.instrument}
              onChange={(e) => setFilters({ ...filters, instrument: e.target.value })}
            />
            <Input
              placeholder="Location"
              value={filters.location}
              onChange={(e) => setFilters({ ...filters, location: e.target.value })}
            />
            <Input
              type="number"
              placeholder="Min budget"
              value={filters.min_budget}
              onChange={(e) => setFilters({ ...filters, min_budget: e.target.value })}
            />
            <Input
              type="number"
              placeholder="Max budget"
              value={filters.max_budget}
              onChange={(e) => setFilters({ ...filters, max_budget: e.target.value })}
            />
            <Select value={filters.sort_by} onValueChange={(v) => setFilters({ ...filters, sort_by: v })}>
              <SelectTrigger>
                <SelectValue placeholder="Sort by" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="newest">Newest</SelectItem>
                <SelectItem value="relevance">Relevance</SelectItem>
                <SelectItem value="budget">Highest Budget</SelectItem>
                <SelectItem value="applications">Fewest Applicants</SelectItem>
                <SelectItem value="popularity">Most Popular</SelectItem>
              </SelectContent>
            </Select>
          </div>
        )}

        {loading ? (
          <div className="flex justify-center py-24">
            <Loader2 className="w-8 h-8 animate-spin text-primary" />
          </div>
        ) : list?.length ? (
          <div className="space-y-4">
            {list.map((job) => (
              <JobCard key={job.id} job={job} alreadyApplied={appliedJobIds.has(job.id)} onApplied={onApplied} />
            ))}
          </div>
        ) : tab === "for-you" ? (
          <div className="text-center py-24 text-muted-foreground">
            No recommendations yet. Complete your profile to get matched with gigs.
          </div>
        ) : (
          <div className="text-center py-24 text-muted-foreground">No open gigs right now. Check back soon!</div>
        )}
      </div>
    </div>
  );
}
