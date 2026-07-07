export default function manifest() {
  return {
    name: "GigPurse",
    short_name: "GigPurse",
    description: "Connecting talented musicians with clients for unforgettable events.",
    start_url: "/",
    display: "standalone",
    background_color: "#ffffff",
    theme_color: "#c2540e",
    icons: [
      { src: "/icons/icon-192.png", sizes: "192x192", type: "image/png" },
      { src: "/icons/icon-512.png", sizes: "512x512", type: "image/png" },
      { src: "/icons/maskable-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" },
    ],
  };
}
