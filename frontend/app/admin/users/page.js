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
import { Loader2, CheckCircle2, XCircle, Search } from "lucide-react";
import AdvancedFilters from "@/components/admin/AdvancedFilters";

const ROLE_COLOR = {
  admin: "bg-violet-500/10 text-violet-600 dark:text-violet-400",
  moderator: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  client: "bg-sky-500/10 text-sky-600 dark:text-sky-400",
  musician: "bg-primary/10 text-primary",
};

const emptyAdvanced = { verified: "any", disabled: "any", from: "", to: "" };

export default function AdminUsers() {
  const [search, setSearch] = useState("");
  const [role, setRole] = useState("all");
  const [advanced, setAdvanced] = useState(emptyAdvanced);
  const { data: users, isLoading } = useQuery({
    queryKey: ["admin-users"],
    queryFn: () => apiGet("/admin/users"),
  });

  const activeAdvancedCount = Object.entries(advanced).filter(([k, v]) =>
    k === "verified" || k === "disabled" ? v !== "any" : Boolean(v),
  ).length;

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return (users || []).filter((u) => {
      if (role !== "all" && u.role !== role) return false;
      if (
        q &&
        !u.name?.toLowerCase().includes(q) &&
        !u.email?.toLowerCase().includes(q)
      )
        return false;
      if (
        advanced.verified !== "any" &&
        Boolean(u.email_verified) !== (advanced.verified === "yes")
      )
        return false;
      if (
        advanced.disabled !== "any" &&
        Boolean(u.disabled) !== (advanced.disabled === "yes")
      )
        return false;
      if (advanced.from && new Date(u.created_at) < new Date(advanced.from))
        return false;
      if (
        advanced.to &&
        new Date(u.created_at) > new Date(advanced.to + "T23:59:59")
      )
        return false;
      return true;
    });
  }, [users, search, role, advanced]);

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
            placeholder="Search name or email"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select value={role} onValueChange={setRole}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All roles</SelectItem>
            <SelectItem value="client">Client</SelectItem>
            <SelectItem value="musician">Talent</SelectItem>
            <SelectItem value="moderator">Moderator</SelectItem>
            <SelectItem value="admin">Admin</SelectItem>
          </SelectContent>
        </Select>
        <AdvancedFilters
          activeCount={activeAdvancedCount}
          onClear={() => setAdvanced(emptyAdvanced)}
        >
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Email verified
            </label>
            <Select
              value={advanced.verified}
              onValueChange={(v) => setAdvanced({ ...advanced, verified: v })}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="any">Any</SelectItem>
                <SelectItem value="yes">Verified</SelectItem>
                <SelectItem value="no">Not verified</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Account status
            </label>
            <Select
              value={advanced.disabled}
              onValueChange={(v) => setAdvanced({ ...advanced, disabled: v })}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="any">Any</SelectItem>
                <SelectItem value="no">Active</SelectItem>
                <SelectItem value="yes">Disabled</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">
              Joined between
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
          {filtered.length} of {users?.length ?? 0}
        </span>
      </div>

      <div className="bg-card rounded-2xl border border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left text-muted-foreground">
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Email</th>
              <th className="px-4 py-3 font-medium">Role</th>
              <th className="px-4 py-3 font-medium">Verified</th>
              <th className="px-4 py-3 font-medium">Joined</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((u) => (
              <tr
                key={u.id}
                className="border-b border-border last:border-0 transition-colors hover:bg-muted/40"
              >
                <td className="px-4 py-3 text-foreground font-medium">
                  {u.name}
                </td>
                <td className="px-4 py-3 text-muted-foreground">{u.email}</td>
                <td className="px-4 py-3">
                  <span
                    className={`inline-flex px-2.5 py-1 rounded-full text-xs font-medium capitalize ${ROLE_COLOR[u.role] || "bg-muted text-muted-foreground"}`}
                  >
                    {u.role}
                  </span>
                </td>
                <td className="px-4 py-3 text-muted-foreground">
                  {u.email_verified ? (
                    <CheckCircle2 className="w-4 h-4 text-emerald-500" />
                  ) : (
                    <XCircle className="w-4 h-4 text-muted-foreground/50" />
                  )}
                </td>
                <td className="px-4 py-3 text-muted-foreground">
                  {new Date(u.created_at).toLocaleDateString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!filtered.length && (
          <p className="p-6 text-center text-sm text-muted-foreground">
            {users?.length ? "No users match your filters." : "No users yet."}
          </p>
        )}
      </div>
    </div>
  );
}
