"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { hasInAppHistory } from "@/components/SiteChrome";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import ReviewCard from "@/components/reviews/ReviewCard";
import BookingModal from "@/components/booking/BookingModal";
import ShareLinkButton from "@/components/ShareLinkButton";
import MediaThumb from "@/components/portfolio/MediaThumb";
import PortfolioLightbox from "@/components/portfolio/PortfolioLightbox";
import IconBadge from "@/components/ui/icon-badge";
import { formatMoney } from "@/lib/utils";
import { Loader2, MapPin, MessageCircle, Music, Star, Briefcase, Wallet, UserRound, ArrowLeft } from "lucide-react";

export default function TalentProfileClient({ id }) {
  const { user, isAuthenticated } = useCurrentUser();
  const router = useRouter();
  const [previewItem, setPreviewItem] = useState(null);

  const { data: musician, isLoading } = useQuery({
    queryKey: ["musician", id],
    queryFn: () => apiGet(`/musicians/${id}`),
    enabled: !!id,
  });

  const { data: reviews } = useQuery({
    queryKey: ["reviews", id],
    queryFn: () => apiGet(`/reviews?user_id=${id}`),
    enabled: !!id,
  });

  const { data: averageRating } = useQuery({
    queryKey: ["reviews-average", id],
    queryFn: () => apiGet(`/reviews/average?user_id=${id}`),
    enabled: !!id,
  });

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!musician) {
    return <div className="min-h-screen bg-background flex items-center justify-center text-muted-foreground">Talent not found.</div>;
  }

  const mp = musician.musician_profile || {};
  const canBook = isAuthenticated && user?.role === "client";
  const isOwnProfile = isAuthenticated && user?.id === musician.id;
  const displayName = mp.stage_name || musician.name;
  const portfolio = mp.portfolio || [];
  const featured = portfolio.filter((item) => item.is_featured);
  const rest = portfolio.filter((item) => !item.is_featured);
  const profileURL = typeof window !== "undefined" ? window.location.href : "";

  // router.back() silently lands on a blank page when there's no prior
  // in-app history (a direct link, a fresh tab) — fall back to somewhere
  // useful instead of a dead end.
  function handleBack() {
    if (hasInAppHistory()) {
      router.back();
    } else {
      router.push("/browse");
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-background">
        <div className="max-w-5xl mx-auto px-4 py-3">
          <button
            type="button"
            onClick={handleBack}
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            Back
          </button>
        </div>
      </div>
      <div className="h-16 sm:h-28 bg-primary" />
      <div className="max-w-5xl mx-auto px-4 -mt-8 sm:-mt-12 pb-16">
        <div className="grid lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-card rounded-2xl shadow-sm border border-border p-6">
              <div className="flex items-start justify-between gap-4 flex-wrap">
                <div className="flex items-start gap-5">
                  <div className="w-16 h-16 sm:w-20 sm:h-20 rounded-2xl bg-primary flex items-center justify-center text-primary-foreground text-xl sm:text-2xl font-bold shrink-0 -mt-8 sm:-mt-10 shadow-lg ring-4 ring-card">
                    {(displayName || "?").charAt(0).toUpperCase()}
                  </div>
                  <div className="pt-2">
                    <h1 className="text-2xl font-bold text-foreground">{displayName}</h1>
                    {musician.location && (
                      <p className="text-muted-foreground flex items-center gap-1 mt-1">
                        <MapPin className="w-4 h-4" />
                        {musician.location}
                      </p>
                    )}
                    <div className="flex items-center gap-1 mt-2 text-sm text-foreground">
                      <Star className="w-4 h-4 text-amber-500 fill-amber-500" />
                      {averageRating?.average_rating ? averageRating.average_rating.toFixed(1) : "New"}
                      <span className="text-muted-foreground">· {reviews?.length || 0} reviews</span>
                    </div>
                  </div>
                </div>
                {profileURL && <ShareLinkButton url={profileURL} className="shrink-0" />}
              </div>

              <div className="flex flex-wrap gap-2 mt-6">
                {(mp.genres || []).map((g) => (
                  <Badge key={g} variant="secondary">
                    {g}
                  </Badge>
                ))}
                {(mp.instruments || []).map((i) => (
                  <Badge key={i} variant="outline" className="gap-1">
                    <Music className="w-3 h-3" />
                    {i}
                  </Badge>
                ))}
              </div>
            </div>

            {(musician.bio || isOwnProfile) && (
              <div className="bg-card rounded-2xl shadow-sm border border-border p-6">
                <h2 className="font-semibold text-foreground mb-3 flex items-center gap-1.5">
                  <UserRound className="w-4 h-4 text-primary" />
                  About
                </h2>
                {musician.bio ? (
                  <p className="text-foreground whitespace-pre-line">{musician.bio}</p>
                ) : (
                  <p className="text-muted-foreground text-sm">
                    No bio yet — add one from your profile settings so clients know more about you.
                  </p>
                )}
              </div>
            )}

            {mp.intro_video_url && (
              <div className="bg-card rounded-2xl shadow-sm border border-border p-6">
                <h2 className="font-semibold text-foreground mb-3">Intro Video</h2>
                <a href={mp.intro_video_url} target="_blank" rel="noreferrer" className="text-primary hover:underline break-all">
                  {mp.intro_video_url}
                </a>
              </div>
            )}

            {featured.length > 0 && (
              <div className="bg-card rounded-2xl shadow-sm border border-border p-6">
                <h2 className="font-semibold text-foreground mb-4 flex items-center gap-1.5">
                  <Star className="w-4 h-4 text-amber-500 fill-amber-500" />
                  Featured work
                </h2>
                <div className="grid sm:grid-cols-2 gap-4">
                  {featured.map((item, idx) => (
                    <PortfolioTile key={idx} item={item} onClick={() => setPreviewItem(item)} featured />
                  ))}
                </div>
              </div>
            )}

            {rest.length > 0 && (
              <div className="bg-card rounded-2xl shadow-sm border border-border p-6">
                <h2 className="font-semibold text-foreground mb-4">Portfolio</h2>
                <div className="grid sm:grid-cols-3 gap-3">
                  {rest.map((item, idx) => (
                    <PortfolioTile key={idx} item={item} onClick={() => setPreviewItem(item)} />
                  ))}
                </div>
              </div>
            )}

            <div className="bg-card rounded-2xl shadow-sm border border-border p-6">
              <h2 className="font-semibold text-foreground mb-4">Reviews</h2>
              {reviews?.length ? (
                <div className="space-y-3">
                  {reviews.map((review) => (
                    <ReviewCard key={review.id} review={review} />
                  ))}
                </div>
              ) : (
                <p className="text-muted-foreground text-sm">No reviews yet.</p>
              )}
            </div>
          </div>

          <div>
            <div className="bg-card rounded-2xl shadow-sm border border-border p-6 sticky top-24 space-y-4">
              {musician.completed_contracts > 0 && (
                <div>
                  <p className="text-sm text-muted-foreground mb-2">Track record</p>
                  <div className="grid grid-cols-2 gap-3">
                    <div className="flex items-center gap-2.5">
                      <IconBadge icon={Briefcase} color="bg-primary" size="sm" />
                      <div className="min-w-0">
                        <p className="font-bold text-foreground leading-tight">{musician.completed_contracts}</p>
                        <p className="text-xs text-muted-foreground leading-tight">completed</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2.5">
                      <IconBadge icon={Wallet} color="bg-emerald-500" size="sm" />
                      <div className="min-w-0">
                        <p className="font-bold text-foreground leading-tight truncate">{formatMoney(musician.total_earned)}</p>
                        <p className="text-xs text-muted-foreground leading-tight">earned</p>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {(mp.price_min || mp.price_max) && (
                <div>
                  <p className="text-sm text-muted-foreground">Typical price range</p>
                  <p className="text-2xl font-bold text-foreground">
                    {formatMoney(mp.price_min || 0)}
                    {mp.price_max ? ` - ${formatMoney(mp.price_max)}` : "+"}
                  </p>
                </div>
              )}

              {mp.availability?.length > 0 && (
                <div>
                  <p className="text-sm text-muted-foreground mb-1">Availability</p>
                  <div className="flex flex-wrap gap-1.5">
                    {mp.availability.map((a) => (
                      <Badge key={a} variant="secondary">
                        {a}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {isOwnProfile ? (
                <p className="text-sm text-muted-foreground">This is your public profile.</p>
              ) : (
                <>
                  {canBook ? (
                    <BookingModal
                      targetUserId={musician.id}
                      targetName={displayName}
                      trigger={<Button className="w-full">Request booking</Button>}
                    />
                  ) : isAuthenticated ? (
                    <p className="text-sm text-muted-foreground">Only client accounts can request bookings.</p>
                  ) : (
                    <Link href="/login">
                      <Button className="w-full">Log in to book</Button>
                    </Link>
                  )}

                  {isAuthenticated && (
                    <Link href={`/messages?with=${musician.id}`}>
                      <Button variant="outline" className="w-full gap-2">
                        <MessageCircle className="w-4 h-4" />
                        Message
                      </Button>
                    </Link>
                  )}
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      <PortfolioLightbox item={previewItem} open={!!previewItem} onOpenChange={(o) => !o && setPreviewItem(null)} />
    </div>
  );
}

function PortfolioTile({ item, onClick, featured }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`text-left rounded-xl border border-border overflow-hidden hover:border-primary/40 transition-colors group ${
        featured ? "aspect-video" : "aspect-square"
      }`}
    >
      <div className="relative w-full h-full">
        <MediaThumb item={item} className="rounded-none group-hover:brightness-95 transition-[filter]" />
        <div className="absolute bottom-0 inset-x-0 bg-gradient-to-t from-black/70 to-transparent p-2.5 pt-6">
          <p className="text-white text-xs font-medium truncate">{item.title}</p>
        </div>
      </div>
    </button>
  );
}
