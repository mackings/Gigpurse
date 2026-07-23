"use client";

import { useRef } from "react";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

function formatDigits(raw) {
  // Strip everything but digits — deliberately drops any "." too, since
  // milestone/budget/bid amounts are always whole naira. If a decimal ever
  // slipped through here uncaught, blindly stripping the "." would splice
  // the fractional digits onto the integer (200000.5 -> 2000005), so we
  // truncate at the decimal point first instead.
  const whole = String(raw ?? "").split(".")[0];
  const digits = whole.replace(/[^\d]/g, "");
  if (!digits) return "";
  return Number(digits).toLocaleString("en-NG");
}

// Naira amount input: shows thousands-separated digits ("20,000") as the
// user types instead of a bare number, while the value passed to onChange
// stays a plain digit string so existing parseFloat(...) submit code needs
// no changes. Native <input type="number"> can't render commas at all,
// hence type="text" + inputMode="numeric" here.
export default function CurrencyInput({ value, onChange, className, placeholder = "0", ...props }) {
  const inputRef = useRef(null);

  function handleChange(e) {
    const el = e.target;
    const rawValue = el.value;
    const cursor = el.selectionStart ?? rawValue.length;
    // How many digits sit before the cursor in the string as typed — this
    // is what we need to preserve, not the raw character offset, since
    // reformatting inserts/removes comma characters around it.
    const digitsBeforeCursor = rawValue.slice(0, cursor).replace(/[^\d]/g, "").length;

    const typedDigits = rawValue.replace(/[^\d]/g, "");
    onChange(typedDigits);

    // The DOM value above is stale until React re-renders with the newly
    // formatted string — without this, every reformat (a comma appearing
    // or disappearing) silently shoves the cursor to the end, so typing
    // into the middle of a number drops or misplaces digits.
    requestAnimationFrame(() => {
      const node = inputRef.current;
      if (!node) return;
      const formatted = node.value;
      let seen = 0;
      let pos = formatted.length;
      for (let i = 0; i < formatted.length; i++) {
        if (/\d/.test(formatted[i])) seen++;
        if (seen === digitsBeforeCursor) {
          pos = i + 1;
          break;
        }
      }
      node.setSelectionRange(pos, pos);
    });
  }

  return (
    <div className="relative">
      <span className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground text-sm pointer-events-none">₦</span>
      <Input
        ref={inputRef}
        type="text"
        inputMode="numeric"
        autoComplete="off"
        value={formatDigits(value)}
        onChange={handleChange}
        placeholder={placeholder}
        className={cn("pl-6", className)}
        {...props}
      />
    </div>
  );
}
