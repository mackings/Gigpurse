"use client";

import { motion } from "framer-motion";
import { Search, Send, MessageCircle, PartyPopper, UserPlus, Radar, CalendarCheck, Wallet } from "lucide-react";

const clientSteps = [
  { icon: Search, title: "Browse talent", description: "Search through our curated list of verified talent by genre, instrument, or location." },
  { icon: Send, title: "Request booking", description: "Send a booking request with your event details, date, and budget." },
  { icon: MessageCircle, title: "Chat & confirm", description: "Discuss details with the talent and finalize your booking." },
  { icon: PartyPopper, title: "Enjoy & review", description: "Have an amazing event and leave a review for the community." },
];

const talentSteps = [
  { icon: UserPlus, title: "Create profile", description: "Build your professional profile showcasing your skills, experience, and portfolio." },
  { icon: Radar, title: "Get discovered", description: "Appear in search results and get found by clients looking for your talent." },
  { icon: CalendarCheck, title: "Receive bookings", description: "Review booking requests and communicate with potential clients." },
  { icon: Wallet, title: "Get paid", description: "Complete gigs and receive payments securely through our platform." },
];

function StepList({ steps, kicker }) {
  return (
    <div>
      <div className="text-sm font-semibold text-primary tracking-wide uppercase mb-8">{kicker}</div>
      <div className="relative">
        <div className="absolute left-5 top-2 bottom-2 w-px bg-border" aria-hidden />
        <div className="space-y-2">
          {steps.map((step, index) => (
            <motion.div
              key={step.title}
              initial={{ opacity: 0, y: 10 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: index * 0.08 }}
              className="relative flex gap-5 py-4"
            >
              <div className="relative z-10 w-10 h-10 rounded-full bg-background border-2 border-border flex items-center justify-center shrink-0">
                <step.icon className="w-4.5 h-4.5 text-primary" />
              </div>
              <div className="pt-1.5">
                <h3 className="text-base font-semibold text-foreground mb-1">{step.title}</h3>
                <p className="text-sm text-muted-foreground">{step.description}</p>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </div>
  );
}

export default function HowItWorks() {
  return (
    <section className="py-20 sm:py-28 bg-muted/30 border-y border-border">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <motion.div
          initial={{ opacity: 0, y: 16 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="text-center mb-16"
        >
          <h2 className="text-3xl sm:text-4xl font-bold text-foreground mb-4 tracking-tight">How it works</h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            Whether you&apos;re hiring talent or showcasing your skills, our platform makes it simple.
          </p>
        </motion.div>

        <div className="grid lg:grid-cols-2 gap-16">
          <StepList steps={clientSteps} kicker="For clients" />
          <StepList steps={talentSteps} kicker="For talent" />
        </div>
      </div>
    </section>
  );
}
