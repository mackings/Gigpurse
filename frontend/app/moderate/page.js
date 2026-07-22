"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Loader2, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { requestModeratorLogin, verifyModeratorLogin } from "@/lib/api";
import AuthShell from "@/components/auth/AuthShell";

// Entry point for anyone to go moderate a dispute — no signup, no
// password, no pre-existing account. Just prove ownership of the inbox
// via a one-time code, since this grants access to two people's private
// conversation, screenshots, and voice notes, plus the power to settle
// their escrow. Once verified, this is a normal session — it lands them
// on the same /admin/disputes queue a staff password login would.
export default function ModeratePage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [step, setStep] = useState("email"); // "email" | "code"
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleRequestCode(e) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await requestModeratorLogin(email);
      toast.success("If that's a staff account, a code is on its way — check the inbox.");
      setStep("code");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleVerifyCode(e) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      const { user } = await verifyModeratorLogin({ email, code });
      queryClient.setQueryData(["profile", "me"], user);
      router.push("/admin/disputes");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <AuthShell>
      <Card>
        <CardHeader>
          <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center mb-2">
            <ShieldCheck className="w-5 h-5 text-primary" />
          </div>
          <CardTitle className="text-2xl">Moderator access</CardTitle>
          <CardDescription>
            {step === "email"
              ? "Enter your email — we'll send a one-time code, no account or password needed."
              : `Enter the code we sent to ${email}.`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {step === "email" ? (
            <form onSubmit={handleRequestCode} className="space-y-4">
              <div>
                <Label htmlFor="mod-email">Email</Label>
                <Input
                  id="mod-email"
                  type="email"
                  required
                  autoFocus
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="mt-1.5"
                />
              </div>
              <Button type="submit" disabled={isSubmitting} className="w-full">
                {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : "Send code"}
              </Button>
            </form>
          ) : (
            <form onSubmit={handleVerifyCode} className="space-y-4">
              <div>
                <Label htmlFor="mod-code">Code</Label>
                <Input
                  id="mod-code"
                  required
                  autoFocus
                  inputMode="numeric"
                  placeholder="6-digit code"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  className="mt-1.5"
                />
              </div>
              <Button type="submit" disabled={isSubmitting} className="w-full">
                {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : "Join"}
              </Button>
              <button
                type="button"
                onClick={() => setStep("email")}
                className="w-full text-center text-sm text-muted-foreground hover:text-foreground"
              >
                Use a different email
              </button>
            </form>
          )}
        </CardContent>
      </Card>
    </AuthShell>
  );
}
