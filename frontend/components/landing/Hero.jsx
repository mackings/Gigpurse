"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ArrowRight, Disc3, Mic2, Guitar, Users } from "lucide-react";
import { motion } from "framer-motion";

export default function Hero() {
  return (
    <section className="relative overflow-hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20 sm:py-28">
        <div className="grid lg:grid-cols-2 gap-16 items-center">
          <motion.div
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="text-center lg:text-left"
          >
            <div className="inline-flex items-center gap-2 px-3.5 py-1.5 bg-accent rounded-full text-accent-foreground text-sm font-medium mb-6">
              <Disc3 className="w-3.5 h-3.5" />
              <span>The #1 talent marketplace</span>
            </div>

            <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold text-foreground leading-[1.1] mb-6 tracking-tight">
              Find the perfect <span className="text-primary">musical talent</span> for your event
            </h1>

            <p className="text-lg text-muted-foreground mb-8 max-w-xl mx-auto lg:mx-0">
              Connect with exceptional singers, instrumentalists, bands, and producers. Book verified
              talent for weddings, concerts, church events, and more.
            </p>

            <div className="flex flex-col sm:flex-row gap-3 justify-center lg:justify-start">
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

            <div className="flex items-center justify-center lg:justify-start gap-8 mt-14 pt-8 border-t border-border">
              <div>
                <div className="text-2xl font-bold text-foreground">500+</div>
                <div className="text-sm text-muted-foreground">Talent</div>
              </div>
              <div className="w-px h-10 bg-border" />
              <div>
                <div className="text-2xl font-bold text-foreground">1,200+</div>
                <div className="text-sm text-muted-foreground">Bookings</div>
              </div>
              <div className="w-px h-10 bg-border" />
              <div>
                <div className="text-2xl font-bold text-foreground">4.9</div>
                <div className="text-sm text-muted-foreground">Avg rating</div>
              </div>
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, scale: 0.97 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.5, delay: 0.15 }}
            className="relative hidden lg:block"
          >
            <div className="relative rounded-3xl overflow-hidden border border-border shadow-xl">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src="https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=600&h=700&fit=crop"
                alt="Talent performing"
                className="w-full h-[520px] object-cover"
              />
            </div>

            <motion.div
              animate={{ y: [0, -8, 0] }}
              transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }}
              className="absolute -left-6 top-16 bg-card border border-border rounded-2xl shadow-lg p-4"
            >
              <div className="flex items-center gap-3">
                <div className="w-11 h-11 bg-accent rounded-xl flex items-center justify-center">
                  <Mic2 className="w-5 h-5 text-accent-foreground" />
                </div>
                <div>
                  <div className="text-sm font-semibold text-foreground">Vocalists</div>
                  <div className="text-xs text-muted-foreground">120+ available</div>
                </div>
              </div>
            </motion.div>

            <motion.div
              animate={{ y: [0, 8, 0] }}
              transition={{ duration: 4.5, repeat: Infinity, ease: "easeInOut" }}
              className="absolute -right-6 top-1/2 bg-card border border-border rounded-2xl shadow-lg p-4"
            >
              <div className="flex items-center gap-3">
                <div className="w-11 h-11 bg-accent rounded-xl flex items-center justify-center">
                  <Guitar className="w-5 h-5 text-accent-foreground" />
                </div>
                <div>
                  <div className="text-sm font-semibold text-foreground">Instrumentalists</div>
                  <div className="text-xs text-muted-foreground">200+ available</div>
                </div>
              </div>
            </motion.div>

            <motion.div
              animate={{ y: [0, -6, 0] }}
              transition={{ duration: 5, repeat: Infinity, ease: "easeInOut" }}
              className="absolute left-8 -bottom-6 bg-card border border-border rounded-2xl shadow-lg p-4"
            >
              <div className="flex items-center gap-3">
                <div className="w-11 h-11 bg-accent rounded-xl flex items-center justify-center">
                  <Users className="w-5 h-5 text-accent-foreground" />
                </div>
                <div>
                  <div className="text-sm font-semibold text-foreground">Bands</div>
                  <div className="text-xs text-muted-foreground">50+ available</div>
                </div>
              </div>
            </motion.div>
          </motion.div>
        </div>
      </div>
    </section>
  );
}
