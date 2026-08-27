import Link from "next/link";
import { Card } from "@/components/ui/card";
import { MapPin, Music, Star, ArrowRight } from "lucide-react";
import { formatMoney, initials } from "@/lib/utils";

export default function TalentCard({ musician }) {
  const mp = musician.musician_profile || {};
  const displayName = mp.stage_name || musician.name;
  const hasTags = (mp.genres?.length || 0) > 0 || (mp.instruments?.length || 0) > 0;
  const hasPrice = mp.price_min || mp.price_max;

  return (
    <Link href={`/talent/${musician.id}`} className="group block h-full">
      <Card size="sm" className="h-full pt-0 transition-colors duration-200 hover:border-foreground/15">
        <div className="relative h-40 overflow-hidden rounded-t-xl bg-primary/10">
          {musician.avatar_url ? (
            // Backend-uploaded media is served from a per-environment host
            // (localhost in dev, the Render URL in prod), so it isn't a
            // fixed next/image remotePattern — plain <img> matches how every
            // other uploaded-media surface (MediaThumb, AvatarUpload) does this.
            // eslint-disable-next-line @next/next/no-img-element
            <img src={musician.avatar_url} alt={displayName} className="w-full h-full object-cover" />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-4xl font-bold text-primary">{initials(displayName)}</div>
          )}
          {musician.average_rating > 0 && (
            <div className="absolute top-3 right-3 flex items-center gap-1 bg-background/90 backdrop-blur-sm rounded-full px-2.5 py-1 text-xs font-semibold shadow-sm">
              <Star className="w-3.5 h-3.5 text-status-warning fill-status-warning" />
              {musician.average_rating.toFixed(1)}
            </div>
          )}
        </div>

        <div className="px-(--card-spacing) pt-3 flex flex-col h-[calc(100%-10rem)]">
          <div className="flex items-start justify-between gap-2">
            <h3 className="font-semibold text-foreground truncate group-hover:text-primary transition-colors">{displayName}</h3>
            {hasPrice && (
              <span className="text-sm font-semibold text-foreground shrink-0">
                {formatMoney(mp.price_min || 0)}
                {mp.price_max ? `–${formatMoney(mp.price_max)}` : "+"}
              </span>
            )}
          </div>

          <div className="flex items-center gap-x-3 gap-y-0.5 flex-wrap mt-0.5 text-sm text-muted-foreground">
            {musician.location && (
              <span className="flex items-center gap-1 truncate">
                <MapPin className="w-3.5 h-3.5 shrink-0" />
                {musician.location}
              </span>
            )}
            {mp.experience_years > 0 && <span>{mp.experience_years}+ yrs experience</span>}
          </div>

          {musician.bio && <p className="text-sm text-muted-foreground line-clamp-1 mt-1.5">{musician.bio}</p>}

          {hasTags && (
            <div className="flex flex-wrap gap-1.5 mt-2.5">
              {(mp.genres || []).slice(0, 2).map((g) => (
                <span key={g} className="px-2 py-0.5 bg-accent text-accent-foreground text-xs rounded-full font-medium">
                  {g}
                </span>
              ))}
              {(mp.instruments || []).slice(0, 1).map((i) => (
                <span key={i} className="px-2 py-0.5 bg-muted text-muted-foreground text-xs rounded-full flex items-center gap-1">
                  <Music className="w-3 h-3" />
                  {i}
                </span>
              ))}
            </div>
          )}

          <div className="mt-auto pt-3 border-t border-border flex items-center justify-between">
            <span className="text-xs text-muted-foreground">{musician.average_rating > 0 && `${musician.review_count || 0} reviews`}</span>
            <span className="flex items-center gap-1 text-xs font-medium text-primary">
              View profile
              <ArrowRight className="w-3 h-3 group-hover:translate-x-0.5 transition-transform" />
            </span>
          </div>
        </div>
      </Card>
    </Link>
  );
}
