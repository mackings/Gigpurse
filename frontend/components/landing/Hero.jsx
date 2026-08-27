"use client";

import Link from "next/link";
import Image from "next/image";
import { Button } from "@/components/ui/button";
import { ArrowRight, ArrowUpRight, Star } from "lucide-react";
import { motion } from "framer-motion";

const stats = [
  { value: "500+", label: "Verified talent" },
  { value: "1,200+", label: "Bookings made" },
  { value: "4.9", label: "Average rating" },
];

export default function Hero() {
  return (
    <section className="relative overflow-hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-16 sm:pt-20 pb-20 sm:pb-28">
        <div className="grid lg:grid-cols-[1.05fr_1fr] gap-12 lg:gap-10 items-center">
          <motion.div
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            <div className="inline-flex items-center gap-1.5 text-sm font-semibold text-primary tracking-wide uppercase mb-5">
              <ArrowUpRight className="w-4 h-4" />
              The #1 talent marketplace
            </div>

            <h1 className="text-5xl sm:text-6xl lg:text-[4.5rem] font-bold text-foreground leading-[0.98] mb-6 tracking-tighter text-balance">
              Find the perfect <span className="text-primary">musical talent</span> for your event
            </h1>

            <p className="text-lg text-muted-foreground mb-9 max-w-lg">
              Connect with exceptional singers, instrumentalists, bands, and producers. Book verified
              talent for weddings, concerts, church events, and more.
            </p>

            <div className="flex flex-col sm:flex-row gap-3 mb-14">
              <Link href="/browse">
                <Button size="lg" className="w-full sm:w-auto px-7 text-base rounded-xl">
                  Hire Talent
                  <ArrowRight className="ml-2 w-4 h-4" />
                </Button>
              </Link>
              <Link href="/role-selection">
                <Button size="lg" variant="outline" className="w-full sm:w-auto px-7 text-base rounded-xl">
                  Become a Talent
                </Button>
              </Link>
            </div>

            <div className="flex items-center gap-8">
              {stats.map((stat, i) => (
                <div key={stat.label} className={i > 0 ? "pl-8 border-l border-border" : ""}>
                  <div className="text-3xl font-bold text-foreground tabular-nums tracking-tight">{stat.value}</div>
                  <div className="text-sm text-muted-foreground mt-0.5">{stat.label}</div>
                </div>
              ))}
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, scale: 0.97 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.5, delay: 0.15 }}
            className="relative hidden lg:block h-[600px]"
          >
            <div className="absolute inset-0 [clip-path:polygon(9%_0,100%_0,100%_100%,0_100%)]">
              <Image
                src="/brand/photos/hero-guitarist.jpg"
                alt="Musician performing on stage"
                fill
                priority
                sizes="45vw"
                className="object-cover"
              />
            </div>

            <div className="absolute left-6 bottom-6 right-6 flex items-center gap-3 bg-background/95 backdrop-blur-sm rounded-2xl p-4 shadow-lg">
              <div className="flex -space-x-2 shrink-0">
                {["JP", "AD", "KO"].map((initials) => (
                  <div
                    key={initials}
                    className="w-9 h-9 rounded-full bg-primary/10 text-primary text-xs font-semibold flex items-center justify-center ring-2 ring-background"
                  >
                    {initials}
                  </div>
                ))}
              </div>
              <div className="w-px h-8 bg-border shrink-0" />
              <div className="flex items-center gap-1.5 min-w-0">
                <Star className="w-4 h-4 text-status-warning fill-status-warning shrink-0" />
                <span className="text-sm font-semibold text-foreground">4.9</span>
                <span className="text-sm text-muted-foreground truncate">from 800+ bookings</span>
              </div>
            </div>
          </motion.div>
        </div>
      </div>
    </section>
  );
}
