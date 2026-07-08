"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";

// My Jobs (bids/bookings) now lives under the account settings hub. Kept
// as a redirect so any existing links/bookmarks to the old URL still work.
export default function MyJobsRedirect() {
  const router = useRouter();
  useEffect(() => {
    router.replace("/profile/jobs");
  }, [router]);

  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <Loader2 className="w-8 h-8 animate-spin text-primary" />
    </div>
  );
}
