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
