import Link from "next/link";
import Image from "next/image";

export default function Footer() {
  return (
    <footer className="bg-muted/40 border-t border-border text-muted-foreground">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 py-16">
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-10">
          <div>
            <div className="flex items-center mb-4">
              <Image
                src="/brand/gigpurse-wordmark-teal.png"
                alt="GigPurse"
                width={140}
                height={40}
                className="h-7 w-auto dark:hidden"
              />
              <Image
                src="/brand/gigpurse-wordmark-white-on-black.png"
                alt="GigPurse"
                width={140}
                height={40}
                className="h-7 w-auto hidden dark:block"
              />
            </div>
            <p className="text-sm max-w-xs">Connecting Talent with clients for unforgettable events.</p>
          </div>

          <div>
            <h4 className="font-semibold text-foreground mb-4 text-sm">For Clients</h4>
            <ul className="space-y-2.5 text-sm">
              <li>
                <Link href="/browse" className="hover:text-primary transition-colors">
                  Browse Talent
                </Link>
              </li>
              <li>
                <Link href="/role-selection" className="hover:text-primary transition-colors">
                  How It Works
                </Link>
              </li>
            </ul>
          </div>

          <div>
            <h4 className="font-semibold text-foreground mb-4 text-sm">For Talent</h4>
            <ul className="space-y-2.5 text-sm">
              <li>
                <Link href="/role-selection" className="hover:text-primary transition-colors">
                  Join as Talent
                </Link>
              </li>
            </ul>
          </div>

          <div>
            <h4 className="font-semibold text-foreground mb-4 text-sm">Support</h4>
            <ul className="space-y-2.5 text-sm">
              <li>
                <a href="#" className="hover:text-primary transition-colors">
                  Help Center
                </a>
              </li>
              <li>
                <a href="#" className="hover:text-primary transition-colors">
                  Contact Us
                </a>
              </li>
            </ul>
          </div>
        </div>

        <div className="border-t border-border mt-12 pt-8 text-center text-sm">
          <p>© {new Date().getFullYear()} GigPurse. All rights reserved.</p>
        </div>
      </div>
    </footer>
  );
}
