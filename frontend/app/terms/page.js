export const metadata = {
  title: "Terms and Conditions — GigPurse",
  description: "The terms and conditions for using GigPurse.",
};

const sections = [
  {
    title: "1. Using GigPurse",
    body: "GigPurse connects clients with musical talent for bookings and gigs. You must be at least 18 years old and provide accurate account information to use the platform. You're responsible for all activity under your account.",
  },
  {
    title: "2. Bookings and payments",
    body: "Contracts, milestones, and payments must be completed through GigPurse. Attempting to move a booking off-platform to avoid fees, or sharing personal contact details before a contract is in place, may result in account suspension.",
  },
  {
    title: "3. Conduct",
    body: "Treat other users respectfully. Harassment, discrimination, fraud, or posting misleading job/profile information is not allowed and may lead to removal from the platform.",
  },
  {
    title: "4. Content and portfolios",
    body: "You retain ownership of the media you upload to your portfolio. By uploading it, you grant GigPurse a license to display it on the platform so clients can evaluate your work. Don't upload content you don't have the rights to share.",
  },
  {
    title: "5. Disputes",
    body: "If a booking doesn't go as agreed, either party can open a dispute from the contract page. GigPurse staff will review the details submitted by both sides and work toward a fair resolution.",
  },
  {
    title: "6. Account termination",
    body: "GigPurse may suspend or terminate accounts that violate these terms. You can close your account at any time by contacting support.",
  },
  {
    title: "7. Changes to these terms",
    body: "We may update these terms as the platform evolves. Continued use of GigPurse after a change means you accept the updated terms.",
  },
];

export default function TermsPage() {
  return (
    <div className="min-h-screen bg-background py-12 px-4">
      <div className="max-w-2xl mx-auto">
        <h1 className="text-3xl font-bold text-foreground tracking-tight">Terms and Conditions</h1>
        <p className="text-muted-foreground mt-2">Last updated 2026-07-10.</p>

        <div className="mt-8 space-y-6">
          {sections.map((s) => (
            <div key={s.title}>
              <h2 className="font-semibold text-foreground mb-1.5">{s.title}</h2>
              <p className="text-muted-foreground leading-relaxed">{s.body}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
