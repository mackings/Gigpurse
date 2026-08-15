"use client";

import { useMemo, useState } from "react";
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
import AdvancedFilters from "@/components/admin/AdvancedFilters";
import EngagementTable, { isAtRisk } from "@/components/admin/EngagementTable";
import { Loader2, Search } from "lucide-react";

const emptyAdvanced = { minGigs: "", minSpent: "", atRiskOnly: false };

export default function AdminClients() {
  const [search, setSearch] = useState("");
  const [window_, setWindow] = useState("30");
  const [advanced, setAdvanced] = useState(emptyAdvanced);

  const { data: rows, isLoading } = useQuery({
    queryKey: ["admin-client-engagement", window_],
    queryFn: () => apiGet(`/admin/client-engagement?window=${window_}`),
  });

  const windowDays = Number(window_);
  const activeAdvancedCount =
    (advanced.minGigs ? 1 : 0) +
    (advanced.minSpent ? 1 : 0) +
    (advanced.atRiskOnly ? 1 : 0);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return (rows || []).filter((r) => {
      if (
        q &&
        !r.name?.toLowerCase().includes(q) &&
        !r.email?.toLowerCase().includes(q)
      )
        return false;
      if (advanced.minGigs && r.gigs_count < Number(advanced.minGigs))
        return false;
      if (advanced.minSpent && r.financial_total < Number(advanced.minSpent))
        return false;
      if (advanced.atRiskOnly && !isAtRisk(r, windowDays)) return false;
      return true;
    });
  }, [rows, search, advanced, windowDays]);

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-xs">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search name or email"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select value={window_} onValueChange={setWindow}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="7">Last 7 days</SelectItem>
            <SelectItem value="30">Last 30 days</SelectItem>
            <SelectItem value="90">Last 90 days</SelectItem>
          </SelectContent>
        </Select>
        <AdvancedFilters
          activeCount={activeAdvancedCount}
          onClear={() => setAdvanced(emptyAdvanced)}
        >
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Minimum gigs hired
            </label>
            <Input
              type="number"
              placeholder="0"
              value={advanced.minGigs}
              onChange={(e) =>
                setAdvanced({ ...advanced, minGigs: e.target.value })
              }
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Minimum total spent (₦)
            </label>
            <Input
              type="number"
              placeholder="0"
              value={advanced.minSpent}
              onChange={(e) =>
                setAdvanced({ ...advanced, minSpent: e.target.value })
              }
            />
          </div>
          <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer pt-1">
            <input
              type="checkbox"
              checked={advanced.atRiskOnly}
              onChange={(e) =>
                setAdvanced({ ...advanced, atRiskOnly: e.target.checked })
              }
              className="accent-primary"
            />
            At-risk only (no activity in the window)
          </label>
        </AdvancedFilters>
        <span className="text-sm text-muted-foreground ml-auto shrink-0">
          {filtered.length} of {rows?.length ?? 0}
        </span>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-24">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
        </div>
      ) : (
        <EngagementTable
          rows={filtered}
          windowDays={windowDays}
          financialLabel="Total spent"
          showAvgPerMonth
          emptyLabel={
            rows?.length
              ? "No clients match your filters."
              : "No clients on the platform yet."
          }
        />
      )}
    </div>
  );
}
