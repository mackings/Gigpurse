"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { Mic2, Guitar, Users, Music2, Headphones, Church } from "lucide-react";

const categories = [
  { icon: Mic2, name: "Vocalists", description: "Solo singers, choirs, backup vocals", count: "120+", image: "https://images.unsplash.com/photo-1516280440614-37939bbacd81?w=400&h=300&fit=crop", type: "vocalist" },
  { icon: Guitar, name: "Instrumentalists", description: "Pianists, guitarists, drummers & more", count: "200+", image: "https://images.unsplash.com/photo-1511192336575-5a79af67a629?w=400&h=300&fit=crop", type: "instrumentalist" },
  { icon: Users, name: "Bands", description: "Full bands for any occasion", count: "50+", image: "https://images.unsplash.com/photo-1598387993441-a364f854c3e1?w=400&h=300&fit=crop", type: "band" },
  { icon: Church, name: "Church Talent", description: "Worship leaders & church bands", count: "80+", image: "https://images.unsplash.com/photo-1509824227185-9c5a01ceba0d?w=400&h=300&fit=crop", type: "church_musician" },
  { icon: Music2, name: "Producers", description: "Music producers & composers", count: "40+", image: "https://images.unsplash.com/photo-1598488035139-bdbb2231ce04?w=400&h=300&fit=crop", type: "producer" },
  { icon: Headphones, name: "DJs", description: "Professional DJs for events", count: "60+", image: "https://images.unsplash.com/photo-1571266028243-e4733b0f0bb0?w=400&h=300&fit=crop", type: "dj" },
];

export default function Categories() {
  return (
    <section className="py-20 sm:py-28">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <motion.div
          initial={{ opacity: 0, y: 16 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="text-center mb-16"
        >
          <h2 className="text-3xl sm:text-4xl font-bold text-foreground mb-4 tracking-tight">Explore categories</h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            Find the perfect Talent for your needs from our diverse categories.
          </p>
        </motion.div>

        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {categories.map((category, index) => (
            <motion.div
              key={category.name}
              initial={{ opacity: 0, y: 16 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: index * 0.06 }}
            >
              <Link href={`/browse?talent_type=${category.type}`}>
                <div className="group bg-card rounded-2xl overflow-hidden border border-border hover:border-primary/40 hover:shadow-md transition-all duration-200">
                  <div className="relative h-40 overflow-hidden">
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img
                      src={category.image}
                      alt={category.name}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
                    />
                    <div className="absolute top-3 left-3 w-10 h-10 bg-background/90 backdrop-blur-sm rounded-xl flex items-center justify-center shadow-sm">
                      <category.icon className="w-5 h-5 text-primary" />
                    </div>
                    <div className="absolute bottom-3 right-3 px-2.5 py-1 bg-background/90 backdrop-blur-sm rounded-full text-foreground text-xs font-medium shadow-sm">
                      {category.count}
                    </div>
                  </div>
                  <div className="p-4">
                    <h3 className="font-semibold text-foreground mb-1 group-hover:text-primary transition-colors">
                      {category.name}
                    </h3>
                    <p className="text-sm text-muted-foreground">{category.description}</p>
                  </div>
                </div>
              </Link>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
