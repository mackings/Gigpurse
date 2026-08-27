"use client";

import Link from "next/link";
import Image from "next/image";
import { motion } from "framer-motion";
import { Mic2, Guitar, Users, Music2, Headphones, Church, ArrowRight } from "lucide-react";

const featured = { icon: Users, name: "Bands", description: "Full bands for any occasion — from wedding sets to full concert lineups", count: "50+", image: "/brand/photos/cat-bands.jpg", type: "band" };

const categories = [
  { icon: Mic2, name: "Vocalists", description: "Solo singers, choirs, backup vocals", count: "120+", image: "/brand/photos/cat-vocalists.jpg", type: "vocalist" },
  { icon: Guitar, name: "Instrumentalists", description: "Pianists, guitarists, drummers & more", count: "200+", image: "/brand/photos/cat-instrumentalists.jpg", type: "instrumentalist" },
  { icon: Church, name: "Church Talent", description: "Worship leaders & church bands", count: "80+", image: "/brand/photos/cat-church.jpg", type: "church_musician" },
  { icon: Music2, name: "Producers", description: "Music producers & composers", count: "40+", image: "/brand/photos/cat-producers.jpg", type: "producer" },
  { icon: Headphones, name: "DJs", description: "Professional DJs for events", count: "60+", image: "/brand/photos/cat-djs.jpg", type: "dj" },
];

function Tile({ category, className, imageClassName }) {
  return (
    <Link href={`/browse?talent_type=${category.type}`} className={className}>
      <div className="group relative h-full rounded-2xl overflow-hidden border border-border hover:border-primary/40 transition-colors duration-200">
        <div className={`relative overflow-hidden ${imageClassName}`}>
          <Image
            src={category.image}
            alt={category.name}
            fill
            sizes="(min-width: 1024px) 33vw, (min-width: 640px) 50vw, 100vw"
            className="object-cover group-hover:scale-105 transition-transform duration-500"
          />
          <div className="absolute top-3 left-3 w-10 h-10 bg-background/90 backdrop-blur-sm rounded-xl flex items-center justify-center shadow-sm">
            <category.icon className="w-5 h-5 text-primary" />
          </div>
          <div className="absolute bottom-3 right-3 px-2.5 py-1 bg-background/90 backdrop-blur-sm rounded-full text-foreground text-xs font-medium shadow-sm">
            {category.count}
          </div>
        </div>
        <div className="p-4 bg-card">
          <h3 className="font-semibold text-foreground mb-1 group-hover:text-primary transition-colors">
            {category.name}
          </h3>
          <p className="text-sm text-muted-foreground">{category.description}</p>
        </div>
      </div>
    </Link>
  );
}

export default function Categories() {
  return (
    <section className="py-20 sm:py-28">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <motion.div
          initial={{ opacity: 0, y: 16 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 mb-12"
        >
          <div>
            <h2 className="text-3xl sm:text-4xl font-bold text-foreground mb-4 tracking-tight">Explore categories</h2>
            <p className="text-lg text-muted-foreground max-w-2xl">
              Find the perfect talent for your needs from our diverse categories.
            </p>
          </div>
          <Link
            href="/browse"
            className="hidden sm:inline-flex items-center gap-1.5 text-sm font-semibold text-primary shrink-0 mb-1"
          >
            View all talent
            <ArrowRight className="w-4 h-4" />
          </Link>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 16 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="mb-5"
        >
          <Tile category={featured} className="block" imageClassName="h-56 lg:h-72" />
        </motion.div>

        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {categories.map((category, index) => (
            <motion.div
              key={category.name}
              initial={{ opacity: 0, y: 16 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: (index + 1) * 0.06 }}
            >
              <Tile category={category} className="block h-full" imageClassName="h-40" />
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
