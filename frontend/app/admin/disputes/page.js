"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import StatusBadge from "@/components/ui/status-badge";
import IconBadge from "@/components/ui/icon-badge";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import AdvancedFilters from "@/components/admin/AdvancedFilters";
import { Loader2, ShieldAlert, ChevronRight, Search } from "lucide-react";

const STATUS_COLOR = {
  open: "bg-rose-500",
  resolved: "bg-emerald-500",
  closed: "bg-muted-foreground",
};
const emptyAdvanced = { moderator: "any", from: "", to: "" };

export default function AdminDisputes() {
  const [status, setStatus] = useState("open");
  const [search, setSearch] = useState("");
  const [advanced, setAdvanced] = useState(emptyAdvanced);

  const { data: disputes, isLoading } = useQuery({
    queryKey: ["admin-disputes", status],
    queryFn: () => apiGet(`/admin/disputes?status=${status}`),
  });

  const activeAdvancedCount = Object.entries(advanced).filter(([k, v]) =>
    k === "moderator" ? v !== "any" : Boolean(v),
  ).length;

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return (disputes || []).filter((d) => {
      if (
        q &&
        !d.reason?.toLowerCase().includes(q) &&
        !d.contract_id?.toLowerCase().includes(q)
      )
        return false;
      if (advanced.moderator === "assigned" && !d.moderator_id) return false;
      if (advanced.moderator === "unassigned" && d.moderator_id) return false;
      if (advanced.from && new Date(d.created_at) < new Date(advanced.from))
        return false;
      if (
        advanced.to &&
        new Date(d.created_at) > new Date(advanced.to + "T23:59:59")
      )
        return false;
      return true;
    });
  }, [disputes, search, advanced]);

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-xs">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search reason or contract"
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
            <SelectItem value="open">Open</SelectItem>
            <SelectItem value="resolved">Resolved</SelectItem>
            <SelectItem value="closed">Closed</SelectItem>
          </SelectContent>
        </Select>
        <AdvancedFilters
          activeCount={activeAdvancedCount}
          onClear={() => setAdvanced(emptyAdvanced)}
        >
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Moderator
            </label>
            <Select
              value={advanced.moderator}
              onValueChange={(v) => setAdvanced({ ...advanced, moderator: v })}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="any">Any</SelectItem>
                <SelectItem value="assigned">Assigned</SelectItem>
                <SelectItem value="unassigned">Unassigned</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Opened between
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
          {filtered.length} of {disputes?.length ?? 0}
        </span>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-24">
          <Loader2 className="w-8 h-8 animate-spin text-primary" />
        </div>
      ) : filtered.length ? (
        <div className="space-y-3">
          {filtered.map((d) => (
            <Link
              key={d.id}
              href={`/admin/disputes/${d.id}`}
              className="group flex items-start justify-between gap-4 bg-card rounded-xl border border-border p-4 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-rose-500/30"
            >
              <div className="flex items-start gap-3 min-w-0">
                <IconBadge
                  icon={ShieldAlert}
                  color={STATUS_COLOR[d.status] || "bg-muted-foreground"}
                  size="sm"
                />
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <p className="font-medium text-foreground">{d.reason}</p>
                    <StatusBadge status={d.status} />
                    {!d.moderator_id && d.status === "open" && (
                      <span className="text-xs font-medium text-amber-600 dark:text-amber-400">
                        No moderator yet
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground mt-1">
                    Contract: {d.contract_title || d.contract_id}
                  </p>
                  {d.resolution && (
                    <p className="text-sm text-muted-foreground mt-1">
                      Resolution: {d.resolution}
                    </p>
                  )}
                </div>
              </div>
              <ChevronRight className="w-4 h-4 text-muted-foreground shrink-0 mt-1" />
            </Link>
          ))}
        </div>
      ) : (
        <p className="text-center text-sm text-muted-foreground py-24">
          {disputes?.length
            ? "No disputes match your filters."
            : "No disputes with this status."}
        </p>
      )}
    </div>
  );
}
