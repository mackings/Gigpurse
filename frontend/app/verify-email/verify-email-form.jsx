"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Loader2, CheckCircle } from "lucide-react";
import { toast } from "sonner";
import { verifyEmail, resendVerification } from "@/lib/api";
import AuthShell from "@/components/auth/AuthShell";

export default function VerifyEmailForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [email, setEmail] = useState(searchParams.get("email") || "");
  const [code, setCode] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isResending, setIsResending] = useState(false);
  const [verified, setVerified] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await verifyEmail({ email, code });
      setVerified(true);
      toast.success("Email verified! You can now log in.");
      setTimeout(() => router.push("/login"), 1200);
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleResend() {
    if (!email) {
      toast.error("Enter your email first.");
      return;
    }
    setIsResending(true);
    try {
      await resendVerification(email);
      toast.success("Verification code sent.");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsResending(false);
    }
  }

  return (
    <AuthShell>
      <Card>
        <CardHeader>
          <CardTitle className="text-2xl">Verify your email</CardTitle>
          <CardDescription>Enter the 6-digit code we sent to your email address.</CardDescription>
        </CardHeader>
        <CardContent>
          {verified ? (
            <div className="flex flex-col items-center gap-3 py-6 text-center">
              <CheckCircle className="w-12 h-12 text-emerald-500" />
              <p className="text-foreground">Email verified. Redirecting to login...</p>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <Label htmlFor="email">Email</Label>
                <Input id="email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} className="mt-1.5" />
              </div>
              <div>
                <Label htmlFor="code">Verification code</Label>
                <Input
                  id="code"
                  required
                  maxLength={6}
                  inputMode="numeric"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  className="mt-1.5 tracking-widest text-center text-lg"
                  placeholder="123456"
                />
              </div>
              <Button type="submit" disabled={isSubmitting} className="w-full">
                {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : "Verify Email"}
              </Button>
              <Button type="button" variant="ghost" onClick={handleResend} disabled={isResending} className="w-full">
                {isResending ? <Loader2 className="w-4 h-4 animate-spin" /> : "Resend code"}
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </AuthShell>
  );
}
