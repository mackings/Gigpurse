"use client";

import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

function formatDigits(raw) {
  const digits = String(raw ?? "").replace(/[^\d]/g, "");
  if (!digits) return "";
  return Number(digits).toLocaleString("en-NG");
}

// Naira amount input: shows thousands-separated digits ("20,000") as the
// user types instead of a bare number, while the value passed to onChange
// stays a plain digit string so existing parseFloat(...) submit code needs
// no changes. Native <input type="number"> can't render commas at all,
// hence type="text" + inputMode="numeric" here.
export default function CurrencyInput({ value, onChange, className, placeholder = "0", ...props }) {
  return (
    <div className="relative">
      <span className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground text-sm pointer-events-none">₦</span>
      <Input
        type="text"
        inputMode="numeric"
        autoComplete="off"
        value={formatDigits(value)}
        onChange={(e) => onChange(e.target.value.replace(/[^\d]/g, ""))}
        placeholder={placeholder}
        className={cn("pl-6", className)}
        {...props}
      />
    </div>
  );
}
