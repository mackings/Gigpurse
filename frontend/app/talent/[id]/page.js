import { API_URL } from "@/lib/backend";
import TalentProfileClient from "./talent-profile-client";

async function fetchMusician(id) {
  try {
    const res = await fetch(`${API_URL}/musicians/${id}`, { cache: "no-store" });
    const envelope = await res.json();
    if (!envelope?.success) return null;
    return envelope.data;
  } catch {
    return null;
  }
}

// Server-rendered so a shared talent link actually looks good when it lands
// on WhatsApp/Twitter/iMessage — real title, bio, and a featured portfolio
// image instead of a generic "GigPurse" card.
export async function generateMetadata({ params }) {
  const { id } = await params;
  const musician = await fetchMusician(id);
  if (!musician) {
    return { title: "Talent not found — GigPurse" };
  }

  const mp = musician.musician_profile || {};
  const name = mp.stage_name || musician.name;
  const portfolio = mp.portfolio || [];
  const featured = portfolio.find((p) => p.is_featured) || portfolio[0];
  const image = featured?.thumbnail_url || (featured?.media_type === "image" ? featured.url : undefined);
  const description =
    musician.bio || (mp.genres?.length ? `${name} on GigPurse — ${mp.genres.join(", ")}` : `${name}'s talent profile on GigPurse`);

  return {
    title: `${name} — GigPurse`,
    description,
    openGraph: {
      title: name,
      description,
      type: "profile",
      images: image ? [{ url: image }] : undefined,
    },
    twitter: {
      card: image ? "summary_large_image" : "summary",
      title: name,
      description,
      images: image ? [image] : undefined,
    },
  };
}

export default async function TalentProfilePage({ params }) {
  const { id } = await params;
  return <TalentProfileClient id={id} />;
}
