"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Mic2, Briefcase, Building2, ArrowRight } from "lucide-react";
import { motion } from "framer-motion";

const roles = [
  {
    id: "client",
    icon: Briefcase,
    title: "I want to hire Talent",
    description: "Find and book talented performers for your events, weddings, parties, and more.",
  },
  {
    id: "musician",
    icon: Mic2,
    title: "I am a Talent",
    description: "Showcase your skills, get discovered, and book gigs with clients looking for Talent.",
  },
];

export default function RoleSelection() {
  const router = useRouter();
  const [selectedRole, setSelectedRole] = useState(null);

  function handleContinue() {
    if (!selectedRole) return;
    router.push(`/signup?role=${selectedRole}`);
  }

  return (
    <div className="min-h-screen bg-background relative flex items-center justify-center p-4 overflow-hidden">
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_70%_50%_at_50%_0%,var(--accent),transparent)] opacity-60" />
      <div className="relative max-w-2xl w-full">
        <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} className="text-center mb-10">
          <h1 className="text-3xl sm:text-4xl font-bold text-foreground mb-3 tracking-tight">
            Welcome! How will you use GigPurse?
          </h1>
          <p className="text-lg text-muted-foreground">Select your role to get started with your personalized experience.</p>
        </motion.div>

        <div className="grid sm:grid-cols-3 gap-5 mb-8">
          {roles.map((role, index) => (
            <motion.div key={role.id} initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: index * 0.08 }}>
              <button
                onClick={() => setSelectedRole(role.id)}
                className={`w-full h-full text-left p-6 rounded-2xl border-2 transition-all duration-200 ${
                  selectedRole === role.id
                    ? "bg-accent border-primary shadow-md"
                    : "bg-card border-border hover:border-primary/40 hover:shadow-sm"
                }`}
              >
                <div className="w-12 h-12 rounded-xl bg-primary flex items-center justify-center mb-4">
                  <role.icon className="w-6 h-6 text-primary-foreground" />
                </div>
                <h3 className="text-lg font-semibold text-foreground mb-2">{role.title}</h3>
                <p className="text-muted-foreground text-sm">{role.description}</p>
              </button>
            </motion.div>
          ))}
          <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: roles.length * 0.08 }}>
            <div
              className="relative w-full h-full text-left p-6 rounded-2xl border-2 border-dashed border-border bg-card/50 opacity-70 cursor-not-allowed"
              title="Organization accounts are coming soon"
            >
              <span className="absolute top-4 right-4 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
                Coming soon
              </span>
              <div className="w-12 h-12 rounded-xl bg-muted-foreground/40 flex items-center justify-center mb-4">
                <Building2 className="w-6 h-6 text-background" />
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">I am an organization</h3>
              <p className="text-muted-foreground text-sm">
                Set up a recurring hiring plan with top-rated Talent, vetted and selected by GigPurse.
              </p>
            </div>
          </motion.div>
        </div>

        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.25 }} className="text-center">
          <Button onClick={handleContinue} disabled={!selectedRole} size="lg" className="px-8 text-base rounded-xl">
            Continue
            <ArrowRight className="ml-2 w-4 h-4" />
          </Button>
        </motion.div>
      </div>
    </div>
  );
}
