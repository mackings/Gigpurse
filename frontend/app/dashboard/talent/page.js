"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import StatCard from "@/components/dashboard/StatCard";
import BookingRequestsList from "@/components/booking/BookingRequestsList";
import { Loader2, Briefcase, Star, Clock, CheckCircle2, ChevronRight } from "lucide-react";

export default function TalentDashboard() {
  const { user } = useCurrentUser();
  const { data, isLoading } = useQuery({
    queryKey: ["talent-dashboard"],
    queryFn: () => apiGet("/talent/dashboard"),
  });

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  const profileComplete = !!user?.musician_profile?.stage_name;

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-6xl mx-auto px-4 py-12">
        <div className="mb-8 flex items-center justify-between flex-wrap gap-4">
          <div>
            <h1 className="text-3xl font-bold text-foreground tracking-tight">Welcome back, {user?.name}</h1>
            <p className="text-muted-foreground">Here&apos;s what&apos;s happening with your gigs.</p>
          </div>
          <div className="flex gap-3">
            <Link href="/profile/portfolio">
              <Button variant="outline">Portfolio</Button>
            </Link>
            <Link href="/profile/jobs">
              <Button variant="outline">My jobs</Button>
            </Link>
            <Link href="/jobs">
              <Button>Find gigs</Button>
            </Link>
          </div>
        </div>

        {!profileComplete && (
          <div className="mb-8 p-4 rounded-xl bg-amber-500/10 border border-amber-500/30 flex items-center justify-between gap-4 flex-wrap">
            <p className="text-amber-700 dark:text-amber-300 text-sm">Complete your profile to start getting discovered by clients.</p>
            <Link href="/onboarding">
              <Button size="sm" variant="outline" className="border-amber-500/40 text-amber-700 dark:text-amber-300 hover:bg-amber-500/10">
                Complete profile
              </Button>
            </Link>
          </div>
        )}

        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-10">
          <StatCard icon={Clock} label="Pending applications" value={data?.pending_applications?.length || 0} color="bg-amber-500" />
          <StatCard icon={Briefcase} label="Active jobs" value={data?.active_jobs?.length || 0} color="bg-primary" />
          <StatCard icon={CheckCircle2} label="Completed jobs" value={data?.completed_jobs?.length || 0} color="bg-emerald-500" />
          <StatCard icon={Star} label="Average rating" value={data?.average_rating ? data.average_rating.toFixed(1) : "New"} color="bg-sky-500" />
        </div>

        <div id="booking-requests" className="mb-10 scroll-mt-24">
          <h2 className="font-semibold text-foreground mb-4">Booking requests</h2>
          <BookingRequestsList />
        </div>

        <div className="grid lg:grid-cols-2 gap-8">
          <div>
            <h2 className="font-semibold text-foreground mb-4">Your contracts</h2>
            <div className="space-y-3">
              {data?.contracts?.length ? (
                data.contracts.map((contract) => (
                  <Link
                    key={contract.id}
                    href={`/contracts/${contract.id}`}
                    className="bg-card rounded-xl border border-border p-4 flex items-center justify-between gap-3 hover:border-primary/40 transition-colors"
                  >
                    <div>
                      <p className="font-medium text-foreground">{contract.title || "Contract"}</p>
                      <p className="text-sm text-muted-foreground">{contract.price} · <span className="capitalize">{contract.status}</span></p>
                    </div>
                    <ChevronRight className="w-4 h-4 text-muted-foreground shrink-0" />
                  </Link>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">No contracts yet.</p>
              )}
            </div>
          </div>

          <div>
            <h2 className="font-semibold text-foreground mb-4">Recommended for you</h2>
            <div className="space-y-3">
              {data?.recommended_jobs?.length ? (
                data.recommended_jobs.map((job) => (
                  <Link key={job.id} href="/jobs" className="block bg-card rounded-xl border border-border p-4 hover:border-primary/40 transition-colors">
                    <p className="font-medium text-foreground">{job.title}</p>
                    <div className="flex gap-2 mt-1">
                      <Badge variant="secondary">{job.genre}</Badge>
                      <Badge variant="outline">{job.instrument}</Badge>
                    </div>
                  </Link>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">No recommendations yet.</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
