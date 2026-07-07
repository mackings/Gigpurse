"use client";

import { motion } from "framer-motion";

const clientSteps = [
  { number: "01", title: "Browse musicians", description: "Search through our curated list of verified musicians by genre, instrument, or location." },
  { number: "02", title: "Request booking", description: "Send a booking request with your event details, date, and budget." },
  { number: "03", title: "Chat & confirm", description: "Discuss details with the musician and finalize your booking." },
  { number: "04", title: "Enjoy & review", description: "Have an amazing event and leave a review for the community." },
];

const talentSteps = [
  { number: "01", title: "Create profile", description: "Build your professional profile showcasing your skills, experience, and portfolio." },
  { number: "02", title: "Get discovered", description: "Appear in search results and get found by clients looking for your talent." },
  { number: "03", title: "Receive bookings", description: "Review booking requests and communicate with potential clients." },
  { number: "04", title: "Get paid", description: "Complete gigs and receive payments securely through our platform." },
];

function StepList({ steps, badgeLabel }) {
  return (
    <div>
      <div className="inline-flex items-center px-3.5 py-1.5 rounded-full text-sm font-semibold mb-8 bg-accent text-accent-foreground">
        {badgeLabel}
      </div>
      <div className="space-y-0">
        {steps.map((step, index) => (
          <motion.div
            key={step.title}
            initial={{ opacity: 0, y: 10 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ delay: index * 0.08 }}
            className="flex gap-5 py-5 border-b border-border last:border-0"
          >
            <span className="text-2xl font-bold text-muted-foreground/40 tabular-nums shrink-0 w-10">{step.number}</span>
            <div>
              <h3 className="text-base font-semibold text-foreground mb-1">{step.title}</h3>
              <p className="text-sm text-muted-foreground">{step.description}</p>
            </div>
          </motion.div>
        ))}
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
          <StepList steps={clientSteps} badgeLabel="For clients" />
          <StepList steps={talentSteps} badgeLabel="For musicians" />
        </div>
      </div>
    </section>
  );
}
