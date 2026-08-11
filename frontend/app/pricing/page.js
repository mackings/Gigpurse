import Link from "next/link";
import { backendFetch } from "@/lib/backend";
import { Button } from "@/components/ui/button";
import IconBadge from "@/components/ui/icon-badge";
import { formatMoney } from "@/lib/utils";
import { Ban, CheckCircle2, HandCoins, Percent, ShieldCheck, Wallet } from "lucide-react";

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

export default async function PricingPage() {
  const { talent_commission_rate: talentRate, client_service_fee_rate: clientRate } = await getPricing();

  const exampleAgreedPrice = 100000;
  const talentTakeHome = exampleAgreedPrice * (1 - talentRate);
  const platformFee = exampleAgreedPrice * (talentRate + clientRate);
  const clientTotal = exampleAgreedPrice + exampleAgreedPrice * clientRate;

  return (
    <div className="min-h-screen bg-background py-16 px-4">
      <div className="max-w-3xl mx-auto">
        <div className="text-center">
          <h1 className="text-3xl sm:text-4xl font-bold text-foreground tracking-tight">Simple, transparent pricing</h1>
          <p className="text-muted-foreground mt-3 max-w-xl mx-auto">
            One flat rate for everyone. No subscriptions, no listing fees, nothing charged until money actually
            moves — and every payment sits in secure escrow until the client releases it.
          </p>
        </div>

        <div className="grid sm:grid-cols-2 gap-4 mt-10">
          <div className="bg-card rounded-2xl border border-border p-6">
            <IconBadge icon={HandCoins} color="bg-primary" />
            <h2 className="font-semibold text-foreground text-lg mt-4">For talent</h2>
            <p className="text-3xl font-bold text-foreground mt-1 tabular-nums">
              Keep {Math.round((1 - talentRate) * 100)}%
            </p>
            <p className="text-sm text-muted-foreground mt-2 leading-relaxed">
              GigPurse takes a {Math.round(talentRate * 100)}% commission when a milestone is released to you —
              that&apos;s the only fee you&apos;ll ever see. No cost to create a profile, apply to gigs, or list your work.
            </p>
          </div>

          <div className="bg-card rounded-2xl border border-border p-6">
            <IconBadge icon={ShieldCheck} color="bg-violet-500" />
            <h2 className="font-semibold text-foreground text-lg mt-4">For clients</h2>
            <p className="text-3xl font-bold text-foreground mt-1 tabular-nums">
              +{Math.round(clientRate * 100)}% service fee
            </p>
            <p className="text-sm text-muted-foreground mt-2 leading-relaxed">
              On top of the price you agree with your talent, GigPurse adds a {Math.round(clientRate * 100)}%
              service fee to cover secure escrow, payment processing, and dispute support if anything goes wrong.
            </p>
          </div>
        </div>

        <div className="bg-card rounded-2xl border border-border p-6 mt-4">
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
          <div className="inline-flex items-center gap-1.5 text-xs text-muted-foreground bg-muted/40 rounded-full px-3 py-1 mb-4">
            <Percent className="w-3 h-3" />
            Rates shown reflect what&apos;s actually charged on every payment
          </div>
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
