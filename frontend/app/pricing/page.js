import Link from "next/link";
import { backendFetch } from "@/lib/backend";
import { Button } from "@/components/ui/button";
import IconBadge from "@/components/ui/icon-badge";
import { Accordion, AccordionItem, AccordionTrigger, AccordionContent } from "@/components/ui/accordion";
import { formatMoney } from "@/lib/utils";
import { Ban, CheckCircle2, HandCoins, Minus, ShieldCheck, Wallet } from "lucide-react";

export const metadata = {
  title: "Pricing — GigPurse",
  description: "GigPurse's fee structure for clients and talent.",
};

// Server-rendered, straight from the backend's own rates (see
// usecase/pricing.go) rather than a hardcoded copy that could drift —
// backendFetch talks to the Go API directly, no auth needed for this route.
async function getPricing() {
  const fallback = { talent_commission_rate: 0.1, client_service_fee_rate: 0.05 };
  try {
    const { envelope } = await backendFetch("/pricing");
    return envelope?.success ? envelope.data : fallback;
  } catch {
    return fallback;
  }
}

// Grouped like Upwork's "Key Features" table, but for GigPurse's real,
// single-rate, two-sided structure — a feature applies to whichever side
// (or both) actually gets it, not "Basic vs Plus" tiers that don't exist.
const FEATURE_GROUPS = [
  {
    title: "Find the right match",
    rows: [
      {
        name: "Marketplace access",
        detail: "Browse every verified talent profile or open gig on the platform, no cost to look around.",
        client: true,
        talent: true,
      },
      {
        name: "Ratings & reviews after every contract",
        detail: "Both sides leave a review once a contract completes, building a track record either can point to.",
        client: true,
        talent: true,
      },
      {
        name: "Post a gig or apply directly",
        detail: "Clients post as many gigs as they like; talent applies with a proposal and price bid — neither costs anything.",
        client: true,
        talent: true,
      },
      {
        name: "Direct booking requests",
        detail: "Skip the job post entirely and book someone you already know the fit for.",
        client: true,
        talent: true,
      },
    ],
  },
  {
    title: "Get paid, safely",
    rows: [
      {
        name: "Secure escrow on every milestone",
        detail: "Funds sit with our payment partner the moment a milestone is funded — never released without the client's go-ahead.",
        client: true,
        talent: true,
      },
      {
        name: "Milestone-based payments",
        detail: "Break a contract into milestones and pay only for approved work, not the whole budget upfront.",
        client: true,
        talent: true,
      },
      {
        name: "Fast bank payouts",
        detail: "Once a milestone is released, payout goes straight to your linked bank account.",
        client: false,
        talent: true,
      },
      {
        name: "Auto-release after 48h",
        detail: "Request release on a funded milestone, and it pays out automatically if the client doesn't respond within 48 hours.",
        client: false,
        talent: true,
      },
      {
        name: "Moderator-reviewed dispute resolution",
        detail: "If something goes wrong, a moderator reviews the case and decides how funds are settled — not either party alone.",
        client: true,
        talent: true,
      },
    ],
  },
  {
    title: "Stay in sync",
    rows: [
      {
        name: "Real-time in-app messaging",
        detail: "Message your client or talent directly, with milestone cards you can act on right inside the chat.",
        client: true,
        talent: true,
      },
      {
        name: "Contract workspace",
        detail: "Every milestone, payment, and status update for a contract lives in one place.",
        client: true,
        talent: true,
      },
      {
        name: "Notifications for every update",
        detail: "In-app and email alerts the moment a booking, milestone, or message needs your attention.",
        client: true,
        talent: true,
      },
    ],
  },
];

function FeatureCell({ included }) {
  return included ? (
    <CheckCircle2 className="w-[18px] h-[18px] text-primary shrink-0" strokeWidth={2} />
  ) : (
    <Minus className="w-[18px] h-[18px] text-muted-foreground/30 shrink-0" strokeWidth={2} />
  );
}

export default async function PricingPage() {
  const { talent_commission_rate: talentRate, client_service_fee_rate: clientRate } = await getPricing();

  const exampleAgreedPrice = 100000;
  const talentTakeHome = exampleAgreedPrice * (1 - talentRate);
  const platformFee = exampleAgreedPrice * (talentRate + clientRate);
  const clientTotal = exampleAgreedPrice + exampleAgreedPrice * clientRate;

  return (
    <div className="min-h-screen bg-background py-16 px-4">
      <div className="max-w-4xl mx-auto">
        <div className="text-center">
          <h1 className="text-3xl sm:text-4xl font-bold text-foreground tracking-tight">Simple, transparent pricing</h1>
          <p className="text-muted-foreground mt-3 max-w-xl mx-auto">
            One flat rate for everyone. No subscriptions, no listing fees, nothing charged until money actually
            moves — and every payment sits in secure escrow until the client releases it.
          </p>
        </div>

        {/* Two cards, Upwork's card layout — but the two sides of GigPurse's
            actual marketplace, not competing tiers. */}
        <div className="grid sm:grid-cols-2 gap-5 mt-10">
          <div className="bg-card rounded-2xl border border-border p-6 flex flex-col">
            <IconBadge icon={ShieldCheck} color="bg-violet-500" />
            <h2 className="text-2xl font-bold text-foreground mt-4">For clients</h2>
            <p className="text-sm text-muted-foreground mt-1">Hire talent for any gig, big or small</p>

            <Button asChild size="lg" className="mt-5" variant="outline">
              <Link href="/role-selection">Get started for free</Link>
            </Button>

            <div className="border-t border-border mt-6 pt-5">
              <p className="text-3xl font-bold text-foreground tabular-nums">+{Math.round(clientRate * 100)}%</p>
              <p className="text-sm text-muted-foreground mt-1">
                service fee, added on top of what you agree with talent
              </p>
              <p className="text-xs text-muted-foreground mt-2">
                For example, a {formatMoney(exampleAgreedPrice)} milestone costs {formatMoney(clientTotal)} in total.
              </p>
            </div>

            <ul className="text-sm text-foreground space-y-2.5 mt-6">
              {["Post unlimited gigs, free", "Pay only for approved work", "Escrow protection on every milestone", "Moderator support if a dispute comes up"].map((f) => (
                <li key={f} className="flex items-start gap-2">
                  <CheckCircle2 className="w-4 h-4 text-primary shrink-0 mt-0.5" />
                  <span>{f}</span>
                </li>
              ))}
            </ul>
          </div>

          <div className="bg-card rounded-2xl border-2 border-primary p-6 flex flex-col relative">
            <span className="absolute -top-3 right-6 bg-primary text-primary-foreground text-xs font-semibold px-3 py-1 rounded-full">
              MOST COMMON
            </span>
            <IconBadge icon={HandCoins} color="bg-primary" />
            <h2 className="text-2xl font-bold text-foreground mt-4">For talent</h2>
            <p className="text-sm text-muted-foreground mt-1">Get booked and paid for your work</p>

            <Button asChild size="lg" className="mt-5">
              <Link href="/role-selection">Get started for free</Link>
            </Button>

            <div className="border-t border-border mt-6 pt-5">
              <p className="text-3xl font-bold text-foreground tabular-nums">Keep {Math.round((1 - talentRate) * 100)}%</p>
              <p className="text-sm text-muted-foreground mt-1">
                {Math.round(talentRate * 100)}% commission, only charged when you&apos;re paid
              </p>
              <p className="text-xs text-muted-foreground mt-2">
                A {formatMoney(exampleAgreedPrice)} milestone pays out {formatMoney(talentTakeHome)} to your bank account.
              </p>
            </div>

            <ul className="text-sm text-foreground space-y-2.5 mt-6">
              {["Free profile, portfolio, and applications", "No fee unless you actually get paid", "Fast bank payouts on release", "Auto-release after 48h of silence"].map((f) => (
                <li key={f} className="flex items-start gap-2">
                  <CheckCircle2 className="w-4 h-4 text-primary shrink-0 mt-0.5" />
                  <span>{f}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* Key Features — Upwork's grouped, expandable comparison table,
            reworked as Client vs Talent instead of Basic vs Plus. */}
        <div className="mt-16">
          <h2 className="text-2xl font-bold text-foreground">Key features</h2>

          <div className="mt-6 rounded-2xl border border-border bg-card overflow-hidden">
            <div className="overflow-x-auto">
              <div className="min-w-[560px]">
                <div className="grid grid-cols-[1fr_88px_88px] items-center px-5 py-4 border-b border-border bg-muted/30">
                  <span />
                  <span className="text-sm font-semibold text-foreground text-center">Client</span>
                  <span className="text-sm font-semibold text-foreground text-center">Talent</span>
                </div>

                {FEATURE_GROUPS.map((group) => (
                  <div key={group.title}>
                    <div className="px-5 pt-5 pb-1">
                      <p className="text-sm font-semibold text-foreground">{group.title}</p>
                    </div>
                    <Accordion type="multiple">
                      {group.rows.map((row) => (
                        <AccordionItem key={row.name} value={row.name} className="px-5">
                          <AccordionTrigger className="hover:no-underline py-3">
                            <div className="grid grid-cols-[1fr_88px_88px] items-center flex-1 gap-2 pr-2">
                              <span className="text-sm text-foreground text-left">{row.name}</span>
                              <span className="flex justify-center">
                                <FeatureCell included={row.client} />
                              </span>
                              <span className="flex justify-center">
                                <FeatureCell included={row.talent} />
                              </span>
                            </div>
                          </AccordionTrigger>
                          <AccordionContent>
                            <p className="text-sm text-muted-foreground pr-8">{row.detail}</p>
                          </AccordionContent>
                        </AccordionItem>
                      ))}
                    </Accordion>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="bg-card rounded-2xl border border-border p-6 mt-16">
          <h2 className="font-semibold text-foreground mb-4">A worked example</h2>
          <p className="text-sm text-muted-foreground mb-5">
            Say a client and talent agree on a {formatMoney(exampleAgreedPrice)} milestone:
          </p>
          <div className="grid sm:grid-cols-3 gap-3">
            <div className="rounded-xl bg-muted/40 p-4">
              <p className="text-xs text-muted-foreground">Client pays</p>
              <p className="text-xl font-bold text-foreground tabular-nums mt-1">{formatMoney(clientTotal)}</p>
              <p className="text-xs text-muted-foreground mt-1">agreed price + {Math.round(clientRate * 100)}% fee</p>
            </div>
            <div className="rounded-xl bg-muted/40 p-4">
              <p className="text-xs text-muted-foreground">Talent receives</p>
              <p className="text-xl font-bold text-foreground tabular-nums mt-1">{formatMoney(talentTakeHome)}</p>
              <p className="text-xs text-muted-foreground mt-1">agreed price − {Math.round(talentRate * 100)}% commission</p>
            </div>
            <div className="rounded-xl bg-muted/40 p-4">
              <p className="text-xs text-muted-foreground">GigPurse earns</p>
              <p className="text-xl font-bold text-foreground tabular-nums mt-1">{formatMoney(platformFee)}</p>
              <p className="text-xs text-muted-foreground mt-1">combined commission + fee</p>
            </div>
          </div>
        </div>

        <div className="bg-card rounded-2xl border border-border p-6 mt-4">
          <h2 className="font-semibold text-foreground mb-4">How escrow works</h2>
          <div className="space-y-4">
            {[
              {
                icon: Wallet,
                title: "Client funds a milestone",
                body: "Money moves into secure escrow, held by GigPurse's payment partner — not released to talent yet.",
              },
              {
                icon: CheckCircle2,
                title: "Talent delivers the work",
                body: "Both sides communicate and track progress right inside the contract.",
              },
              {
                icon: HandCoins,
                title: "Client releases payment",
                body: "Once the client confirms the work is done, the talent's share is sent straight to their bank account.",
              },
            ].map((step) => (
              <div key={step.title} className="flex items-start gap-3">
                <IconBadge icon={step.icon} color="bg-primary" size="sm" />
                <div>
                  <p className="font-medium text-foreground text-sm">{step.title}</p>
                  <p className="text-sm text-muted-foreground">{step.body}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="bg-card rounded-2xl border border-border p-6 mt-4">
          <h2 className="font-semibold text-foreground mb-3 flex items-center gap-2">
            <Ban className="w-4 h-4 text-muted-foreground" />
            No hidden fees
          </h2>
          <ul className="text-sm text-muted-foreground space-y-1.5 list-disc list-inside">
            <li>No subscription or monthly fee for either side</li>
            <li>No fee to post a job or apply to one</li>
            <li>No fee to build or browse a portfolio</li>
            <li>Nothing charged until a milestone is actually funded</li>
          </ul>
        </div>

        <div className="text-center mt-10">
          <div className="flex items-center justify-center gap-3">
            <Button asChild size="lg">
              <Link href="/role-selection">Get started</Link>
            </Button>
            <Button asChild size="lg" variant="outline">
              <Link href="/browse">Browse talent</Link>
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
