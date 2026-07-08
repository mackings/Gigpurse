import Link from "next/link";
import { MapPin, Music, Star } from "lucide-react";
import { formatMoney } from "@/lib/utils";

export default function TalentCard({ musician }) {
  const mp = musician.musician_profile || {};
  return (
    <Link href={`/talent/${musician.id}`}>
      <div className="group bg-card rounded-2xl border border-border hover:border-primary/40 hover:shadow-lg hover:shadow-black/5 hover:-translate-y-0.5 transition-all duration-200 p-5 h-full flex flex-col">
        <div className="flex items-center gap-4 mb-4">
          <div className="w-12 h-12 rounded-full bg-primary flex items-center justify-center text-primary-foreground font-semibold text-lg shrink-0">
            {(mp.stage_name || musician.name || "?").charAt(0).toUpperCase()}
          </div>
          <div className="min-w-0">
            <h3 className="font-semibold text-foreground truncate group-hover:text-primary transition-colors">
              {mp.stage_name || musician.name}
            </h3>
            {musician.location && (
              <p className="text-sm text-muted-foreground flex items-center gap-1 truncate">
                <MapPin className="w-3.5 h-3.5 shrink-0" />
                {musician.location}
              </p>
            )}
          </div>
        </div>

        <div className="flex flex-wrap gap-1.5 mb-3">
          {(mp.genres || []).slice(0, 3).map((g) => (
            <span key={g} className="px-2 py-0.5 bg-accent text-accent-foreground text-xs rounded-full font-medium">
              {g}
            </span>
          ))}
        </div>

        <div className="flex flex-wrap gap-1.5 mb-4">
          {(mp.instruments || []).slice(0, 3).map((i) => (
            <span key={i} className="px-2 py-0.5 bg-muted text-muted-foreground text-xs rounded-full flex items-center gap-1">
              <Music className="w-3 h-3" />
              {i}
            </span>
          ))}
        </div>

        <div className="mt-auto flex items-center justify-between pt-3 border-t border-border">
          <div className="flex items-center gap-1 text-sm text-muted-foreground">
            <Star className="w-4 h-4 text-amber-500 fill-amber-500" />
            {musician.average_rating ? musician.average_rating.toFixed(1) : "New"}
          </div>
          {(mp.price_min || mp.price_max) && (
            <span className="text-sm font-semibold text-foreground">
              {formatMoney(mp.price_min || 0)}
              {mp.price_max ? ` - ${formatMoney(mp.price_max)}` : "+"}
            </span>
          )}
        </div>
      </div>
    </Link>
  );
}
