"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useDirectHires } from "@/hooks/use-direct-hire";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import JobBoardCard from "@/components/jobs/JobBoardCard";
import IconBadge from "@/components/ui/icon-badge";
import StatusBadge from "@/components/ui/status-badge";
import { formatMoney } from "@/lib/utils";
import { Loader2, Search, SlidersHorizontal, Sparkles, Clock3, Heart, Mail, Handshake, LayoutDashboard, CalendarClock, X } from "lucide-react";

const TABS = [
  { value: "best", label: "Best matches", icon: Sparkles },
  { value: "recent", label: "Most recent", icon: Clock3 },
  { value: "saved", label: "Saved jobs", icon: Heart },
  { value: "invites", label: "Invites", icon: Mail },
];

const EMPTY_FILTERS = { query: "", genre: "", instrument: "", location: "", min_budget: "", max_budget: "", sort_by: "newest" };

function activeFilterCount(filters) {
  return ["genre", "instrument", "location", "min_budget", "max_budget"].filter((k) => filters[k]).length;
}

function buildQuery(filters, extra) {
  const params = new URLSearchParams();
  if (filters.query) params.set("query", filters.query);
  if (filters.genre) params.set("genre", filters.genre);
  if (filters.instrument) params.set("instrument", filters.instrument);
  if (filters.location) params.set("location", filters.location);
  if (filters.min_budget) params.set("min_budget", filters.min_budget);
  if (filters.max_budget) params.set("max_budget", filters.max_budget);
  Object.entries(extra || {}).forEach(([k, v]) => params.set(k, v));
  return params.toString();
}

export default function JobBoard() {
  const { user } = useCurrentUser();
  const queryClient = useQueryClient();
  const isMusician = user?.role === "musician";
  const [tab, setTab] = useState("best");
  const [filters, setFilters] = useState(EMPTY_FILTERS);
  const [filterPopoverOpen, setFilterPopoverOpen] = useState(false);

  // Non-musicians (or an unauthenticated visitor) never see the tab bar, so
  // silently treat them as if "Most recent" were selected — "Best matches"
  // is a musician-only, profile-personalized endpoint.
  const effectiveTab = isMusician ? tab : "recent";
  const searchableTab = effectiveTab === "best" || effectiveTab === "recent";

  const { data: recommended, isLoading: bestLoading } = useQuery({
    queryKey: ["jobs", "recommended", filters],
    queryFn: () => apiGet(`/jobs/recommended?${buildQuery(filters)}`),
    enabled: effectiveTab === "best",
  });

  const { data: recentJobs, isLoading: recentLoading } = useQuery({
    queryKey: ["jobs", "recent", filters],
    queryFn: () => apiGet(`/jobs?${buildQuery(filters, { status: "open", sort_by: filters.sort_by || "newest" })}`),
    enabled: effectiveTab === "recent",
  });

  const { data: savedJobs, isLoading: savedLoading } = useQuery({
    queryKey: ["saved-jobs"],
    queryFn: () => apiGet("/jobs/saved"),
    enabled: isMusician,
  });

  const { requests: allRequests, isLoading: invitesLoading } = useDirectHires();
  const invites = (allRequests || []).filter((r) => r.musician_id === user?.id && r.proposed_by !== user?.id);

  const { data: myApplications } = useQuery({
    queryKey: ["applications", "mine"],
    queryFn: () => apiGet("/jobs/applications"),
    enabled: isMusician,
  });

  const appliedJobIds = new Set((myApplications || []).map((a) => a.job_id));
  const savedJobIds = new Set((savedJobs || []).map((j) => j.id));
  const onApplied = () => queryClient.invalidateQueries({ queryKey: ["applications", "mine"] });

  const list = effectiveTab === "best" ? recommended : effectiveTab === "recent" ? recentJobs : effectiveTab === "saved" ? savedJobs : null;
  const loading =
    effectiveTab === "best" ? bestLoading : effectiveTab === "recent" ? recentLoading : effectiveTab === "saved" ? savedLoading : invitesLoading;
  const filterCount = activeFilterCount(filters);

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-5xl mx-auto px-4 py-12">
        <div className="mb-6 flex items-center justify-between flex-wrap gap-4">
          <div>
            <h1 className="text-3xl font-bold text-foreground mb-2 tracking-tight">Find Gigs</h1>
            <p className="text-muted-foreground">Browse, save, and apply to gigs posted by clients.</p>
          </div>
          {isMusician && (
            <Link href="/dashboard/talent">
              <Button variant="outline" size="sm" className="gap-1.5">
                <LayoutDashboard className="w-3.5 h-3.5" />
                My stats
              </Button>
            </Link>
          )}
        </div>

        {isMusician && searchableTab && (
          <div className="flex gap-2 mb-6">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                placeholder="Search for gigs — title or description"
                value={filters.query}
                onChange={(e) => setFilters({ ...filters, query: e.target.value })}
                className="pl-9 h-11 rounded-full"
              />
            </div>
            <Popover open={filterPopoverOpen} onOpenChange={setFilterPopoverOpen}>
              <PopoverTrigger asChild>
                <Button variant="outline" className="gap-1.5 h-11 rounded-full shrink-0">
                  <SlidersHorizontal className="w-4 h-4" />
                  Filters
                  {filterCount > 0 && (
                    <span className="w-5 h-5 rounded-full bg-primary text-primary-foreground text-xs font-semibold flex items-center justify-center">
                      {filterCount}
                    </span>
                  )}
                </Button>
              </PopoverTrigger>
              <PopoverContent align="end" className="w-80 p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <p className="font-medium text-foreground">Filters</p>
                  {filterCount > 0 && (
                    <button
                      type="button"
                      onClick={() => setFilters({ ...EMPTY_FILTERS, query: filters.query })}
                      className="text-xs text-muted-foreground hover:text-foreground flex items-center gap-1"
                    >
                      <X className="w-3 h-3" />
                      Clear
                    </button>
                  )}
                </div>
                <div>
                  <Label className="text-xs text-muted-foreground">Genre</Label>
                  <Input
                    placeholder="e.g. Afrobeats"
                    value={filters.genre}
                    onChange={(e) => setFilters({ ...filters, genre: e.target.value })}
                    className="mt-1"
                  />
                </div>
                <div>
                  <Label className="text-xs text-muted-foreground">Instrument</Label>
                  <Input
                    placeholder="e.g. Guitar"
                    value={filters.instrument}
                    onChange={(e) => setFilters({ ...filters, instrument: e.target.value })}
                    className="mt-1"
                  />
                </div>
                <div>
                  <Label className="text-xs text-muted-foreground">Location</Label>
                  <Input
                    placeholder="e.g. Lagos"
                    value={filters.location}
                    onChange={(e) => setFilters({ ...filters, location: e.target.value })}
                    className="mt-1"
                  />
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <Label className="text-xs text-muted-foreground">Min budget (₦)</Label>
                    <Input
                      type="number"
                      value={filters.min_budget}
                      onChange={(e) => setFilters({ ...filters, min_budget: e.target.value })}
                      className="mt-1"
                    />
                  </div>
                  <div>
                    <Label className="text-xs text-muted-foreground">Max budget (₦)</Label>
                    <Input
                      type="number"
                      value={filters.max_budget}
                      onChange={(e) => setFilters({ ...filters, max_budget: e.target.value })}
                      className="mt-1"
                    />
                  </div>
                </div>
                {tab === "recent" && (
                  <div>
                    <Label className="text-xs text-muted-foreground">Sort by</Label>
                    <Select value={filters.sort_by} onValueChange={(v) => setFilters({ ...filters, sort_by: v })}>
                      <SelectTrigger className="mt-1">
                        <SelectValue placeholder="Sort by" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="newest">Newest</SelectItem>
                        <SelectItem value="budget">Highest Budget</SelectItem>
                        <SelectItem value="applications">Fewest Applicants</SelectItem>
                        <SelectItem value="popularity">Most Popular</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                )}
                <Button size="sm" className="w-full" onClick={() => setFilterPopoverOpen(false)}>
                  Show results
                </Button>
              </PopoverContent>
            </Popover>
          </div>
        )}

        {isMusician && (
          <div className="flex gap-1 mb-8 border-b border-border overflow-x-auto">
            {TABS.map((t) => (
              <button
                key={t.value}
                onClick={() => setTab(t.value)}
                className={`flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px whitespace-nowrap transition-colors ${
                  tab === t.value
                    ? "border-primary text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                }`}
              >
                <t.icon className="w-3.5 h-3.5" />
                {t.label}
                {t.value === "saved" && savedJobs?.length > 0 && (
                  <span className="ml-0.5 text-xs text-muted-foreground">({savedJobs.length})</span>
                )}
                {t.value === "invites" && invites.length > 0 && (
                  <span className="ml-0.5 text-xs text-muted-foreground">({invites.length})</span>
                )}
              </button>
            ))}
          </div>
        )}

        {loading ? (
          <div className="flex justify-center py-24">
            <Loader2 className="w-8 h-8 animate-spin text-primary" />
          </div>
        ) : effectiveTab === "invites" ? (
          invites.length ? (
            <div className="space-y-4">
              {invites.map((req) => (
                <InviteCard key={req.id} req={req} />
              ))}
            </div>
          ) : (
            <div className="text-center py-24 text-muted-foreground">
              No invites yet. Clients can invite you directly from your talent profile.
            </div>
          )
        ) : list?.length ? (
          <div className="space-y-4">
            {list.map((job) => (
              <JobBoardCard
                key={job.id}
                job={job}
                saved={savedJobIds.has(job.id)}
                alreadyApplied={appliedJobIds.has(job.id)}
                onApplied={onApplied}
                showSaveButton={isMusician}
              />
            ))}
          </div>
        ) : effectiveTab === "best" ? (
          <div className="text-center py-24 text-muted-foreground">
            {filters.query || filterCount > 0
              ? "No gigs match your search."
              : "No recommendations yet. Complete your profile to get matched with gigs."}
          </div>
        ) : effectiveTab === "saved" ? (
          <div className="text-center py-24 text-muted-foreground">
            No saved jobs yet — tap the heart on a listing to save it for later.
          </div>
        ) : (
          <div className="text-center py-24 text-muted-foreground">
            {filters.query || filterCount > 0 ? "No gigs match your search." : "No open gigs right now. Check back soon!"}
          </div>
        )}
      </div>
    </div>
  );
}

function InviteCard({ req }) {
  return (
    <Link
      href={`/messages?with=${req.client_id}&booking=${req.id}`}
      className="group block bg-card rounded-2xl border border-border p-5 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/30"
    >
      <div className="flex items-start gap-4">
        <IconBadge icon={Handshake} color="bg-violet-500" />
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-3">
            <h3 className="font-semibold text-foreground truncate">{req.title}</h3>
            <StatusBadge status={req.status} />
          </div>
          <p className="text-muted-foreground mt-1 line-clamp-2">{req.description}</p>
          <div className="flex items-center gap-3 mt-3 text-sm">
            <span className="font-medium text-foreground">{formatMoney(req.price)}</span>
            {req.event_date && (
              <span className="flex items-center gap-1 text-muted-foreground">
                <CalendarClock className="w-3.5 h-3.5" />
                {new Date(req.event_date).toLocaleDateString()}
              </span>
            )}
          </div>
        </div>
      </div>
    </Link>
  );
}
