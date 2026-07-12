import { clsx } from "clsx";
import { twMerge } from "tailwind-merge"

export function cn(...inputs) {
  return twMerge(clsx(inputs));
}

// Every price/budget/bid/balance in the app is a raw Naira number from the
// API (e.g. 23000) — format it consistently as "₦23,000" everywhere instead
// of dumping the bare number. Pass decimals for ledger values (wallet
// balances) that should always show cents.
export function formatMoney(amount, { decimals } = {}) {
  const n = Number(amount);
  if (!Number.isFinite(n)) return amount ?? "";
  const opts =
    decimals != null
      ? { minimumFractionDigits: decimals, maximumFractionDigits: decimals }
      : { maximumFractionDigits: 2 };
  return `₦${n.toLocaleString("en-NG", opts)}`;
}

// Shared "Posted 3 days ago" phrasing used by every job card/detail view.
export function postedAgo(dateStr) {
  if (!dateStr) return null;
  const diffMs = Date.now() - new Date(dateStr).getTime();
  if (!Number.isFinite(diffMs) || diffMs < 0) return null;
  const days = Math.floor(diffMs / 86400000);
  if (days < 1) return "Posted today";
  if (days === 1) return "Posted yesterday";
  if (days < 30) return `Posted ${days} days ago`;
  return `Posted ${new Date(dateStr).toLocaleDateString()}`;
}

export const JOB_DURATION_LABELS = {
  less_than_1_week: "Less than 1 week",
  "1_to_2_weeks": "1 to 2 weeks",
  "1_to_4_weeks": "1 to 4 weeks",
  "1_to_3_months": "1 to 3 months",
  "3_plus_months": "3+ months",
};

export const JOB_EXPERIENCE_LABELS = {
  entry: "Entry level",
  intermediate: "Intermediate",
  expert: "Expert",
};

export const JOB_PROJECT_TYPE_LABELS = {
  one_time: "One-time gig",
  ongoing: "Ongoing",
};
