"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useCurrentUser } from "@/hooks/use-current-user";
import Hero from "@/components/landing/Hero";
import HowItWorks from "@/components/landing/HowItWorks";
import Categories from "@/components/landing/Categories";
import CTA from "@/components/landing/CTA";
import { Loader2 } from "lucide-react";

export default function Home() {
  const router = useRouter();
  const { user, isLoading, isAuthenticated } = useCurrentUser();

  useEffect(() => {
    if (!isAuthenticated) return;
    // Signed-in users have nothing to do on the marketing landing page —
    // send them straight to their actual home base.
    const dashboard = user?.role === "musician" ? "/jobs" : "/dashboard/client";
    router.replace(dashboard);
  }, [isAuthenticated, user, router]);

  if (isLoading || isAuthenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="min-h-screen">
      <Hero />
      <HowItWorks />
      <Categories />
      <CTA />
    </div>
  );
}
