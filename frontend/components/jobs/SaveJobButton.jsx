"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiPost } from "@/lib/api";
import { Heart } from "lucide-react";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

export default function SaveJobButton({ jobId, saved, className }) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () => apiPost(saved ? "/jobs/unsave" : "/jobs/save", { job_id: jobId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["saved-jobs"] });
      toast.success(saved ? "Removed from saved jobs." : "Job saved.");
    },
    onError: (err) => toast.error(err.message),
  });

  return (
    <button
      type="button"
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        mutation.mutate();
      }}
      disabled={mutation.isPending}
      aria-label={saved ? "Unsave job" : "Save job"}
      title={saved ? "Unsave job" : "Save job"}
      className={cn(
        "shrink-0 w-9 h-9 rounded-full flex items-center justify-center transition-colors disabled:opacity-50",
        saved ? "text-rose-500 hover:bg-rose-500/10" : "text-muted-foreground hover:text-rose-500 hover:bg-rose-500/10",
        className
      )}
    >
      <Heart className={cn("w-4 h-4", saved && "fill-current")} />
    </button>
  );
}
