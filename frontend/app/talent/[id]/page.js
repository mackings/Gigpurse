"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import ReviewCard from "@/components/reviews/ReviewCard";
import BookingModal from "@/components/booking/BookingModal";
import { Loader2, MapPin, MessageCircle, Music, Star } from "lucide-react";

export default function TalentProfile() {
  const { id } = useParams();
  const { user, isAuthenticated } = useCurrentUser();

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

  return (
    <div className="min-h-screen bg-background">
      <div className="h-40 bg-primary" />
      <div className="max-w-5xl mx-auto px-4 -mt-14 pb-16">
        <div className="grid lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-card rounded-2xl shadow-sm border border-border p-6">
              <div className="flex items-start gap-5">
                <div className="w-20 h-20 rounded-2xl bg-primary flex items-center justify-center text-primary-foreground text-2xl font-bold shrink-0 -mt-10 shadow-lg ring-4 ring-card">
                  {(mp.stage_name || musician.name || "?").charAt(0).toUpperCase()}
                </div>
                <div className="pt-2">
                  <h1 className="text-2xl font-bold text-foreground">{mp.stage_name || musician.name}</h1>
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

              {musician.bio && <p className="text-foreground mt-6">{musician.bio}</p>}

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

            {mp.intro_video_url && (
              <div className="bg-card rounded-2xl shadow-sm border border-border p-6">
                <h2 className="font-semibold text-foreground mb-3">Intro Video</h2>
                <a href={mp.intro_video_url} target="_blank" rel="noreferrer" className="text-primary hover:underline break-all">
                  {mp.intro_video_url}
                </a>
              </div>
            )}

            {mp.portfolio?.length > 0 && (
              <div className="bg-card rounded-2xl shadow-sm border border-border p-6">
                <h2 className="font-semibold text-foreground mb-4">Portfolio</h2>
                <div className="grid sm:grid-cols-2 gap-4">
                  {mp.portfolio.map((item, idx) => (
                    <a
                      key={idx}
                      href={item.url || item.external_url}
                      target="_blank"
                      rel="noreferrer"
                      className="block p-4 rounded-xl border border-border hover:border-primary/40 transition-colors"
                    >
                      <p className="font-medium text-foreground">{item.title}</p>
                      <p className="text-sm text-muted-foreground">{item.description}</p>
                    </a>
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
              {(mp.price_min || mp.price_max) && (
                <div>
                  <p className="text-sm text-muted-foreground">Typical price range</p>
                  <p className="text-2xl font-bold text-foreground">
                    {mp.price_min || 0}
                    {mp.price_max ? ` - ${mp.price_max}` : "+"}
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
                      targetName={mp.stage_name || musician.name}
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
    </div>
  );
}
