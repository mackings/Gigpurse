import { Piano, Guitar, Drum, Mic2, Music2, Music3, Music4 } from "lucide-react";

// Keyed off the canonical instrument list in app/onboarding/page.js. Lucide
// has no dedicated icons for most orchestral/brass instruments, so several
// entries share a reasonable stand-in rather than all falling back to Guitar.
const INSTRUMENT_ICONS = {
  "piano/keyboard": Piano,
  guitar: Guitar,
  bass: Guitar,
  drums: Drum,
  "talking drum": Drum,
  saxophone: Music2,
  trumpet: Music2,
  violin: Music3,
  cello: Music3,
  flute: Music4,
  shekere: Music4,
  voice: Mic2,
};

export function instrumentIcon(instrument) {
  return INSTRUMENT_ICONS[(instrument || "").toLowerCase()] || Music2;
}
