"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Loader2, MapPin, Check } from "lucide-react";
import { toast } from "sonner";
import { apiGet, apiPut } from "@/lib/api";
import { useCurrentUser } from "@/hooks/use-current-user";

export default function ProfilePage() {
  const router = useRouter();
  const { user: authUser } = useCurrentUser();
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [form, setForm] = useState({ name: "", bio: "", location: "", company_name: "" });

  useEffect(() => {
    if (authUser?.role === "musician") {
      router.replace("/onboarding");
      return;
    }
    if (authUser?.role !== "client") return;
    apiGet("/users/profile")
      .then((user) => {
        setForm({
          name: user.name || "",
          bio: user.bio || "",
          location: user.location || "",
          company_name: user.client_profile?.company_name || "",
        });
      })
      .catch(() => {})
      .finally(() => setIsLoading(false));
  }, [authUser, router]);

  async function handleSubmit(e) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await apiPut("/users/profile", {
        name: form.name,
        bio: form.bio,
        location: form.location,
        client_profile: { company_name: form.company_name },
      });
      toast.success("Profile saved!");
      router.push("/dashboard/client");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setIsSubmitting(false);
    }
  }

  if (!authUser || authUser.role === "musician" || isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Your details</CardTitle>
        <CardDescription>This information is visible to talent you contact or hire.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <Label htmlFor="name">Full name</Label>
            <Input
              id="name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="mt-1.5"
            />
          </div>
          <div>
            <Label htmlFor="company_name">Company / organization (optional)</Label>
            <Input
              id="company_name"
              placeholder="e.g. Sunset Events Co."
              value={form.company_name}
              onChange={(e) => setForm({ ...form, company_name: e.target.value })}
              className="mt-1.5"
            />
          </div>
          <div>
            <Label htmlFor="bio">Bio</Label>
            <Textarea
              id="bio"
              placeholder="Tell talent a bit about you or your organization..."
              value={form.bio}
              onChange={(e) => setForm({ ...form, bio: e.target.value })}
              className="mt-1.5 min-h-[120px]"
            />
          </div>
          <div>
            <Label htmlFor="location">Location</Label>
            <div className="relative mt-1.5">
              <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                id="location"
                placeholder="City, State (e.g., Lagos, Nigeria)"
                value={form.location}
                onChange={(e) => setForm({ ...form, location: e.target.value })}
                className="pl-10"
              />
            </div>
          </div>
          <Button type="submit" disabled={isSubmitting} className="gap-2">
            {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <>Save changes <Check className="w-4 h-4" /></>}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
