"use client";

import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { Loader2, CheckCircle2, XCircle } from "lucide-react";

const ROLE_COLOR = {
  admin: "bg-violet-500/10 text-violet-600 dark:text-violet-400",
  client: "bg-sky-500/10 text-sky-600 dark:text-sky-400",
  musician: "bg-primary/10 text-primary",
};

export default function AdminUsers() {
  const { data: users, isLoading } = useQuery({
    queryKey: ["admin-users"],
    queryFn: () => apiGet("/admin/users"),
  });

  if (isLoading) {
    return (
      <div className="flex justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
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
          {users?.map((u) => (
            <tr key={u.id} className="border-b border-border last:border-0 transition-colors hover:bg-muted/40">
              <td className="px-4 py-3 text-foreground font-medium">{u.name}</td>
              <td className="px-4 py-3 text-muted-foreground">{u.email}</td>
              <td className="px-4 py-3">
                <span className={`inline-flex px-2.5 py-1 rounded-full text-xs font-medium capitalize ${ROLE_COLOR[u.role] || "bg-muted text-muted-foreground"}`}>
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
              <td className="px-4 py-3 text-muted-foreground">{new Date(u.created_at).toLocaleDateString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {!users?.length && <p className="p-6 text-center text-sm text-muted-foreground">No users yet.</p>}
    </div>
  );
}
