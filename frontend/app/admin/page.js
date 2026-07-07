"use client";

import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import StatCard from "@/components/dashboard/StatCard";
import { Loader2, Users, Briefcase, MessageSquare, Handshake, ShieldAlert } from "lucide-react";

export default function AdminOverview() {
  const { data, isLoading } = useQuery({
    queryKey: ["admin-analytics"],
    queryFn: () => apiGet("/admin/analytics"),
  });

  if (isLoading) {
    return (
      <div className="flex justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <StatCard icon={Users} label="Total users" value={data?.total_users ?? 0} color="bg-primary" />
      <StatCard icon={Briefcase} label="Total jobs" value={data?.total_jobs ?? 0} color="bg-sky-500" />
      <StatCard icon={MessageSquare} label="Total messages" value={data?.total_messages ?? 0} color="bg-emerald-500" />
      <StatCard icon={Handshake} label="Total contracts" value={data?.total_contracts ?? 0} color="bg-amber-500" />
      <StatCard icon={ShieldAlert} label="Total disputes" value={data?.total_disputes ?? 0} color="bg-rose-500" />
    </div>
  );
}
