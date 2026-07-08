"use client";

import { useEffect } from "react";

export default function ServiceWorkerRegister() {
  useEffect(() => {
    if (!("serviceWorker" in navigator)) return;

    if (process.env.NODE_ENV !== "production") {
      // A cached service worker actively fights Fast Refresh during
      // development (dev chunk URLs aren't content-hashed, so any
      // cache-first strategy pins the browser to stale JS indefinitely).
      // Unregister any previously-installed one so local dev always runs
      // the code actually on disk.
      navigator.serviceWorker.getRegistrations().then((regs) => {
        regs.forEach((reg) => reg.unregister());
      });
      return;
    }

    navigator.serviceWorker.register("/sw.js").catch(() => {});
  }, []);

  return null;
}
