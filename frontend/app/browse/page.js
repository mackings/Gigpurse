"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import TalentCard from "@/components/talent/TalentCard";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Loader2, Search, SlidersHorizontal, X } from "lucide-react";

const EMPTY_FILTERS = { genre: "", instrument: "", location: "", sort_by: "" };

function activeFilterCount(filters) {
  return Object.values(filters).filter(Boolean).length;
}

function buildQuery(filters) {
  const params = new URLSearchParams();
  if (filters.genre) params.set("genre", filters.genre);
  if (filters.instrument) params.set("instrument", filters.instrument);
  if (filters.location) params.set("location", filters.location);
  if (filters.sort_by) params.set("sort_by", filters.sort_by);
  return params.toString();
}

function TalentCardSkeleton() {
  return (
    <Card size="sm" className="pt-0 animate-pulse">
      <div className="h-40 bg-muted rounded-t-xl" />
      <div className="px-(--card-spacing) pt-3 space-y-3">
        <div className="h-4 w-2/3 bg-muted rounded" />
        <div className="h-3 w-1/3 bg-muted rounded" />
        <div className="flex gap-1.5">
          <div className="h-5 w-14 bg-muted rounded-full" />
          <div className="h-5 w-14 bg-muted rounded-full" />
        </div>
      </div>
    </Card>
  );
}

export default function BrowseTalents() {
  const [filters, setFilters] = useState(EMPTY_FILTERS);
  const [filterPopoverOpen, setFilterPopoverOpen] = useState(false);
  const filterCount = activeFilterCount(filters);

  const { data: musicians, isLoading } = useQuery({
    queryKey: ["musicians", filters],
    queryFn: () => apiGet(`/musicians?${buildQuery(filters)}`),
  });

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-[1600px] mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-foreground mb-2 tracking-tight">Browse Talent</h1>
          <p className="text-muted-foreground">Find the perfect talent for your next event.</p>
        </div>

        <div className="flex gap-2 mb-8">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Search by genre — e.g. Afrobeats"
              value={filters.genre}
              onChange={(e) => setFilters({ ...filters, genre: e.target.value })}
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
                    onClick={() => setFilters({ ...EMPTY_FILTERS, genre: filters.genre })}
                    className="text-xs text-muted-foreground hover:text-foreground flex items-center gap-1"
                  >
                    <X className="w-3 h-3" />
                    Clear
                  </button>
                )}
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
              <div>
                <Label className="text-xs text-muted-foreground">Sort by</Label>
                <Select value={filters.sort_by} onValueChange={(v) => setFilters({ ...filters, sort_by: v })}>
                  <SelectTrigger className="mt-1 w-full">
                    <SelectValue placeholder="Sort by" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="rating">Top Rated</SelectItem>
                    <SelectItem value="experience">Most Experienced</SelectItem>
                    <SelectItem value="newest">Newest</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Button size="sm" className="w-full" onClick={() => setFilterPopoverOpen(false)}>
                Show results
              </Button>
            </PopoverContent>
          </Popover>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-5">
            {Array.from({ length: 8 }).map((_, i) => (
              <TalentCardSkeleton key={i} />
            ))}
          </div>
        ) : musicians?.length ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-5">
            {musicians.map((musician) => (
              <TalentCard key={musician.id} musician={musician} />
            ))}
          </div>
        ) : (
          <div className="text-center py-24 text-muted-foreground">No Talent match your search yet.</div>
        )}
      </div>
    </div>
  );
}
