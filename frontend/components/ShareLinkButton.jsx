"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Link2, Check } from "lucide-react";
import { toast } from "sonner";

// url should be absolute (e.g. built with window.location.origin) so
// whatever it's pasted into (WhatsApp, a bio link, an email) works standalone.
export default function ShareLinkButton({ url, label = "Copy link", className, variant = "outline", size = "sm" }) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      toast.success("Link copied!");
      setTimeout(() => setCopied(false), 1800);
    } catch {
      toast.error("Couldn't copy the link — copy it from the address bar instead.");
    }
  }

  return (
    <Button type="button" variant={variant} size={size} onClick={handleCopy} className={className}>
      {copied ? <Check className="w-3.5 h-3.5" /> : <Link2 className="w-3.5 h-3.5" />}
      {copied ? "Copied" : label}
    </Button>
  );
}
