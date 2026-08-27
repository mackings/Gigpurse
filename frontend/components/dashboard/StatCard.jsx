export default function StatCard({ icon: Icon, label, value, color }) {
  return (
    <div className="bg-card rounded-xl border border-border p-6 transition-colors hover:border-foreground/15">
      <div className="flex items-center justify-between mb-5">
        <span className="text-sm font-medium text-muted-foreground">{label}</span>
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center shrink-0 ${color}`}>
          <Icon className="w-5 h-5 text-white" />
        </div>
      </div>
      <div className="text-3xl font-bold text-foreground tabular-nums tracking-tight">{value}</div>
    </div>
  );
}
