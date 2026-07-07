"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import TalentCard from "@/components/talent/TalentCard";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Loader2, Search } from "lucide-react";

function buildQuery(filters) {
  const params = new URLSearchParams();
  if (filters.genre) params.set("genre", filters.genre);
  if (filters.instrument) params.set("instrument", filters.instrument);
  if (filters.location) params.set("location", filters.location);
  if (filters.sort_by) params.set("sort_by", filters.sort_by);
  return params.toString();
}

export default function BrowseTalents() {
  const [filters, setFilters] = useState({ genre: "", instrument: "", location: "", sort_by: "" });

  const { data: musicians, isLoading } = useQuery({
    queryKey: ["musicians", filters],
    queryFn: () => apiGet(`/musicians?${buildQuery(filters)}`),
  });

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-7xl mx-auto px-4 py-12">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-foreground mb-2 tracking-tight">Browse Musicians</h1>
          <p className="text-muted-foreground">Find the perfect talent for your next event.</p>
        </div>

        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-3 mb-8">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Genre (e.g. Afrobeats)"
              className="pl-10"
              value={filters.genre}
              onChange={(e) => setFilters({ ...filters, genre: e.target.value })}
            />
          </div>
          <Input
            placeholder="Instrument (e.g. Guitar)"
            value={filters.instrument}
            onChange={(e) => setFilters({ ...filters, instrument: e.target.value })}
          />
          <Input
            placeholder="Location"
            value={filters.location}
            onChange={(e) => setFilters({ ...filters, location: e.target.value })}
          />
          <Select value={filters.sort_by} onValueChange={(v) => setFilters({ ...filters, sort_by: v })}>
            <SelectTrigger>
              <SelectValue placeholder="Sort by" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="rating">Top Rated</SelectItem>
              <SelectItem value="experience">Most Experienced</SelectItem>
              <SelectItem value="newest">Newest</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {isLoading ? (
          <div className="flex justify-center py-24">
            <Loader2 className="w-8 h-8 animate-spin text-primary" />
          </div>
        ) : musicians?.length ? (
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {musicians.map((musician) => (
              <TalentCard key={musician.id} musician={musician} />
            ))}
          </div>
        ) : (
          <div className="text-center py-24 text-muted-foreground">No musicians match your search yet.</div>
        )}
      </div>
    </div>
  );
}
