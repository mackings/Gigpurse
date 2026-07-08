import { Star } from "lucide-react";
import IconBadge from "@/components/ui/icon-badge";

export default function ReviewCard({ review }) {
  return (
    <div className="group p-4 rounded-xl border border-border bg-card transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-amber-500/30">
      <div className="flex items-start gap-3">
        <IconBadge icon={Star} color="bg-amber-500" size="sm" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1">
            {Array.from({ length: 5 }).map((_, i) => (
              <Star key={i} className={`w-3.5 h-3.5 ${i < review.rating ? "text-amber-500 fill-amber-500" : "text-muted"}`} />
            ))}
          </div>
          {review.comment && <p className="text-foreground text-sm mt-2">{review.comment}</p>}
          <p className="text-xs text-muted-foreground mt-2">{new Date(review.created_at).toLocaleDateString()}</p>
        </div>
      </div>
    </div>
  );
}
