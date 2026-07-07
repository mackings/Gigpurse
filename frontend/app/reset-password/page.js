"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { resetPassword } from "@/lib/api";
import AuthShell from "@/components/auth/AuthShell";

export default function ResetPasswordPage() {
  const router = useRouter();
  const [form, setForm] = useState({ token: "", new_password: "" });
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await resetPassword(form);
      toast.success("Password reset. You can now log in.");
      router.push("/login");
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
          <CardTitle className="text-2xl">Set a new password</CardTitle>
          <CardDescription>Paste the reset token from your email.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <Label htmlFor="token">Reset token</Label>
              <Textarea
                id="token"
                required
                value={form.token}
                onChange={(e) => setForm({ ...form, token: e.target.value })}
                className="mt-1.5 font-mono text-xs"
                rows={3}
              />
            </div>
            <div>
              <Label htmlFor="new_password">New password</Label>
              <Input
                id="new_password"
                type="password"
                required
                minLength={8}
                value={form.new_password}
                onChange={(e) => setForm({ ...form, new_password: e.target.value })}
                className="mt-1.5"
              />
            </div>
            <Button type="submit" disabled={isSubmitting} className="w-full">
              {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : "Reset password"}
            </Button>
          </form>
          <p className="text-sm text-muted-foreground text-center mt-6">
            <Link href="/login" className="text-primary font-medium hover:underline">
              Back to login
            </Link>
          </p>
        </CardContent>
      </Card>
    </AuthShell>
  );
}
