import Link from "next/link";
import Image from "next/image";

export default function AuthShell({ children }) {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="max-w-md w-full">
        <Link href="/" className="flex items-center justify-center mb-8">
          <Image
            src="/brand/gigpurse-wordmark-teal.png"
            alt="GigPurse"
            width={160}
            height={44}
            priority
            className="h-8 w-auto dark:hidden"
          />
          <Image
            src="/brand/gigpurse-wordmark-white-on-black.png"
            alt="GigPurse"
            width={160}
            height={44}
            priority
            className="h-8 w-auto hidden dark:block"
          />
        </Link>
        {children}
      </div>
    </div>
  );
}
