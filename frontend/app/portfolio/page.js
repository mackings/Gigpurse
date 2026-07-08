"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";

// Portfolio now lives under the account settings hub. Kept as a redirect
// so any existing links/bookmarks to the old URL still land somewhere.
export default function PortfolioRedirect() {
  const router = useRouter();
  useEffect(() => {
    router.replace("/profile/portfolio");
  }, [router]);

  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <Loader2 className="w-8 h-8 animate-spin text-primary" />
    </div>
  );
}
