import { Star } from "lucide-react";

export default function ReviewCard({ review }) {
  return (
    <div className="p-4 rounded-xl border border-border bg-card">
      <div className="flex items-center gap-1 mb-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Star key={i} className={`w-4 h-4 ${i < review.rating ? "text-amber-500 fill-amber-500" : "text-muted"}`} />
        ))}
      </div>
      {review.comment && <p className="text-foreground text-sm">{review.comment}</p>}
      <p className="text-xs text-muted-foreground mt-2">{new Date(review.created_at).toLocaleDateString()}</p>
    </div>
  );
}
