"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ArrowRight, Sparkles } from "lucide-react";
import { motion } from "framer-motion";

export default function CTA() {
  return (
    <section className="py-20 sm:py-28">
      <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
        <motion.div
          initial={{ opacity: 0, y: 16 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="relative overflow-hidden rounded-3xl bg-foreground text-background px-8 py-16 sm:px-16 sm:py-20 text-center"
        >
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 bg-background/10 rounded-full text-sm font-medium mb-6">
            <Sparkles className="w-3.5 h-3.5 text-primary" />
            <span>Join thousands of happy users</span>
          </div>

          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-6 tracking-tight">
            Ready to find your perfect match?
          </h2>

          <p className="text-lg text-background/70 mb-10 max-w-2xl mx-auto">
            Whether you need to hire Talent for your event or you&apos;re ready to showcase your own talent, our
            platform connects you with the right people.
          </p>

          <div className="flex flex-col sm:flex-row gap-3 justify-center">
            <Link href="/browse">
              <Button size="lg" className="w-full sm:w-auto px-7 text-base rounded-xl">
                Hire Talent
                <ArrowRight className="ml-2 w-4 h-4" />
              </Button>
            </Link>
            <Link href="/role-selection">
              <Button
                size="lg"
                variant="outline"
                className="w-full sm:w-auto px-7 text-base rounded-xl border-background/20 text-background bg-transparent hover:bg-background/10 hover:text-background"
              >
                Join as Talent
              </Button>
            </Link>
          </div>
        </motion.div>
      </div>
    </section>
  );
}
