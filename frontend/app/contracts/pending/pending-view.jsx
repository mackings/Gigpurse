"use client";

import { useEffect } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Loader2, CheckCircle2, Clock, RefreshCw } from "lucide-react";

const POLL_INTERVAL_MS = 4000;
// After this many failed checks, stop auto-polling and hand control back to
// the user with a manual "Check again" button — PayPetal's confirmation is
// asynchronous and can genuinely take longer than a user wants to sit on
// this page for; the webhook finishes the job in the background regardless
// of whether this page is still open.
const MAX_AUTO_ATTEMPTS = 8;

// Landing page a client/musician returns to after PayPetal's hosted
// checkout — for both a job hire and a milestone payment (both use the same
// "?reference=" redirect shape). Polls the matching finalize endpoint via
// React Query's refetchInterval, which re-fetches the agreement from
// PayPetal itself rather than trusting anything in the URL.
export default function PendingPaymentView() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const reference = searchParams.get("reference");
  const isMilestone = reference?.startsWith("milestone:");
  const finalizeURL = isMilestone ? "/milestones/fund/finalize" : "/jobs/hire/finalize";

  const finalizeQuery = useQuery({
    queryKey: ["finalize-payment", reference, finalizeURL],
    queryFn: () => apiPost(finalizeURL, { reference }),
    enabled: !!reference,
    retry: false,
    refetchInterval: (query) => {
      if (query.state.data || query.state.fetchFailureCount >= MAX_AUTO_ATTEMPTS) return false;
      return POLL_INTERVAL_MS;
    },
  });

  const contractId = finalizeQuery.data?.contract_id;
  const status = !reference
    ? "missing"
    : finalizeQuery.data
      ? "confirmed"
      : finalizeQuery.isLoading
        ? "checking"
        : "pending";
  const autoPollingExhausted = finalizeQuery.failureCount >= MAX_AUTO_ATTEMPTS;

  // If this page loaded inside the payment popup (see lib/payment-popup.js),
  // let the opener know as soon as payment's confirmed and close itself —
  // the user should never have to notice they were in a separate window.
  useEffect(() => {
    if (status !== "confirmed" || !window.opener) return;
    window.opener.postMessage({ source: "gigpurse-payment", status: "confirmed" }, window.location.origin);
    const timer = setTimeout(() => window.close(), 1200);
    return () => clearTimeout(timer);
  }, [status]);

  return (
    <div className="min-h-screen bg-background flex items-center justify-center px-4">
      <div className="max-w-md w-full">
        <Card>
          <CardHeader className="text-center">
            {status === "confirmed" ? (
              <>
                <CheckCircle2 className="w-10 h-10 text-emerald-500 mx-auto mb-2" />
                <CardTitle className="text-2xl">Payment confirmed</CardTitle>
                <CardDescription>
                  {isMilestone ? "Escrow has been funded for this milestone." : "Your hire is confirmed — the gig is now active."}
                </CardDescription>
              </>
            ) : status === "missing" ? (
              <>
                <CardTitle className="text-2xl">Nothing to confirm</CardTitle>
                <CardDescription>This page needs a payment reference — you may have landed here directly.</CardDescription>
              </>
            ) : (
              <>
                <Clock className="w-10 h-10 text-amber-500 mx-auto mb-2" />
                <CardTitle className="text-2xl">Confirming your payment</CardTitle>
                <CardDescription>
                  {status === "pending"
                    ? "PayPetal hasn't confirmed this payment yet — this can take a moment. We'll keep checking, and you'll also get a notification the moment it's confirmed."
                    : "Checking with PayPetal..."}
                </CardDescription>
              </>
            )}
          </CardHeader>
          <CardContent className="flex flex-col items-center gap-3">
            {status === "checking" && <Loader2 className="w-6 h-6 animate-spin text-primary" />}

            {status === "confirmed" && (
              <Button asChild className="w-full">
                <Link href={contractId ? `/contracts/${contractId}` : "/dashboard/client"}>View contract</Link>
              </Button>
            )}

            {status === "pending" && (
              <>
                {!autoPollingExhausted ? (
                  <Loader2 className="w-6 h-6 animate-spin text-primary" />
                ) : (
                  <Button variant="outline" onClick={() => finalizeQuery.refetch()} className="gap-1.5">
                    <RefreshCw className="w-3.5 h-3.5" />
                    Check again
                  </Button>
                )}
                <Button variant="ghost" onClick={() => router.push("/dashboard/client")} className="text-sm">
                  I&apos;ll check back later
                </Button>
              </>
            )}

            {status === "missing" && (
              <Button asChild variant="outline">
                <Link href="/dashboard/client">Go to dashboard</Link>
              </Button>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
