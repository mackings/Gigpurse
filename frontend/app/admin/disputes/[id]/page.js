"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import DisputeChatRoom from "@/components/disputes/DisputeChatRoom";
import { ArrowLeft } from "lucide-react";

export default function AdminDisputeDetail() {
  const { id } = useParams();

  return (
    <div className="h-[calc(100vh-10rem)] min-h-[500px] flex flex-col rounded-2xl border border-border overflow-hidden">
      <Link
        href="/admin/disputes"
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground px-4 pt-3"
      >
        <ArrowLeft className="w-4 h-4" />
        Back to disputes
      </Link>
      <DisputeChatRoom disputeId={id} />
    </div>
  );
}
