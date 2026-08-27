import "@fontsource-variable/bricolage-grotesque";
import "./globals.css";
import { Providers } from "./providers";
import SiteChrome from "@/components/SiteChrome";

export const metadata = {
  title: "GigPurse",
  description: "Connecting Talent with clients for unforgettable events.",
  icons: {
    icon: [
      { url: "/icons/icon-192.png", sizes: "192x192", type: "image/png" },
      { url: "/icons/icon-512.png", sizes: "512x512", type: "image/png" },
    ],
    apple: "/icons/apple-touch-icon.png",
  },
  appleWebApp: {
    capable: true,
    statusBarStyle: "default",
    title: "GigPurse",
  },
};

export const viewport = {
  themeColor: "#ec5e10",
};

export default function RootLayout({ children }) {
  return (
    <html lang="en" className="h-full antialiased" suppressHydrationWarning>
      <body className="min-h-full flex flex-col font-sans bg-background text-foreground" suppressHydrationWarning>
        <Providers>
          <SiteChrome>{children}</SiteChrome>
        </Providers>
      </body>
    </html>
  );
}
