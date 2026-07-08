export default function StatCard({ icon: Icon, label, value, color }) {
  return (
    <div className="group relative overflow-hidden bg-card rounded-2xl border border-border p-5 transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-primary/20 hover:-translate-y-0.5">
      <div className={`absolute -top-6 -right-6 w-24 h-24 rounded-full ${color} opacity-[0.08] blur-2xl transition-opacity duration-300 group-hover:opacity-[0.16]`} />
      <div
        className={`relative w-11 h-11 rounded-xl flex items-center justify-center mb-3 ${color} shadow-sm transition-transform duration-200 ease-out group-hover:scale-110 group-hover:rotate-3`}
      >
        <Icon className="w-5 h-5 text-white" />
      </div>
      <div className="relative text-2xl font-bold text-foreground tabular-nums">{value}</div>
      <div className="relative text-sm text-muted-foreground">{label}</div>
    </div>
  );
}
