"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { Button } from "@/components/ui/button";
import ClientJobCard from "@/components/jobs/ClientJobCard";
import BookingRequestsList from "@/components/booking/BookingRequestsList";
import StatCard from "@/components/dashboard/StatCard";
import { Loader2, Briefcase, Handshake, Star, ChevronRight } from "lucide-react";

export default function ClientDashboard() {
  const { user } = useCurrentUser();

  const { data: jobs, isLoading: jobsLoading } = useQuery({
    queryKey: ["client-jobs", user?.id],
    queryFn: () => apiGet(`/jobs?client_id=${user.id}`),
    enabled: !!user?.id,
  });

  const { data: contracts, isLoading: contractsLoading } = useQuery({
    queryKey: ["contracts"],
    queryFn: () => apiGet("/contracts"),
  });

  const activeContracts = contracts?.filter((c) => c.status === "active") || [];
  const completedContracts = contracts?.filter((c) => c.status === "completed") || [];

  if (!user) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-6xl mx-auto px-4 py-12">
        <div className="mb-8 flex items-center justify-between flex-wrap gap-4">
          <div>
            <h1 className="text-3xl font-bold text-foreground tracking-tight">Welcome back, {user.name}</h1>
            <p className="text-muted-foreground">Manage your gigs and bookings.</p>
          </div>
          <div className="flex gap-3">
            <Link href="/profile">
              <Button variant="outline">Edit profile</Button>
            </Link>
            <Link href="/browse">
              <Button variant="outline">Browse Talent</Button>
            </Link>
            <Link href="/jobs/post">
              <Button>Post a gig</Button>
            </Link>
          </div>
        </div>

        <div className="grid sm:grid-cols-3 gap-4 mb-10">
          <StatCard icon={Briefcase} label="Jobs posted" value={jobs?.length || 0} color="bg-primary" />
          <StatCard icon={Handshake} label="Active contracts" value={activeContracts.length} color="bg-sky-500" />
          <StatCard icon={Star} label="Completed" value={completedContracts.length} color="bg-emerald-500" />
        </div>

        <div className="grid lg:grid-cols-2 gap-8">
          <div>
            <h2 className="font-semibold text-foreground mb-4">Your job postings</h2>
            {jobsLoading ? (
              <Loader2 className="w-6 h-6 animate-spin text-primary" />
            ) : jobs?.length ? (
              <div className="space-y-3">
                {jobs.map((job) => (
                  <ClientJobCard key={job.id} job={job} />
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">You haven&apos;t posted any jobs yet.</p>
            )}

            <h2 className="font-semibold text-foreground mb-4 mt-8">Booking requests</h2>
            <BookingRequestsList />
          </div>

          <div>
            <h2 className="font-semibold text-foreground mb-4">Active contracts</h2>
            {contractsLoading ? (
              <Loader2 className="w-6 h-6 animate-spin text-primary" />
            ) : activeContracts.length ? (
              <div className="space-y-3 mb-8">
                {activeContracts.map((contract) => (
                  <Link
                    key={contract.id}
                    href={`/contracts/${contract.id}`}
                    className="bg-card rounded-xl border border-border p-4 flex items-center justify-between gap-3 hover:border-primary/40 transition-colors"
                  >
                    <div>
                      <p className="font-medium text-foreground">{contract.title || "Contract"}</p>
                      <p className="text-sm text-muted-foreground">{contract.price}</p>
                    </div>
                    <ChevronRight className="w-4 h-4 text-muted-foreground shrink-0" />
                  </Link>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground mb-8">No active contracts.</p>
            )}

            <h2 className="font-semibold text-foreground mb-4">Completed contracts</h2>
            {completedContracts.length ? (
              <div className="space-y-3">
                {completedContracts.map((contract) => (
                  <Link
                    key={contract.id}
                    href={`/contracts/${contract.id}`}
                    className="bg-card rounded-xl border border-border p-4 flex items-center justify-between gap-3 hover:border-primary/40 transition-colors"
                  >
                    <div>
                      <p className="font-medium text-foreground">{contract.title || "Contract"}</p>
                      <p className="text-sm text-muted-foreground">{contract.price}</p>
                    </div>
                    <ChevronRight className="w-4 h-4 text-muted-foreground shrink-0" />
                  </Link>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No completed contracts yet.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
