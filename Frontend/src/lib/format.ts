export function formatSalary(min?: number, max?: number) {
  if (!min && !max) return "Salary not disclosed";
  const fmt = (n: number) =>
    n >= 100000 ? `₹${(n / 100000).toFixed(1).replace(/\.0$/, "")}L` : `₹${n.toLocaleString("en-IN")}`;
  if (min && max) return `${fmt(min)} – ${fmt(max)} / yr`;
  return fmt((min ?? max)!);
}

export function formatDate(iso?: string) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString("en-IN", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

export function formatRelativeDate(iso?: string) {
  if (!iso) return "—";
  const days = Math.round((Date.now() - new Date(iso).getTime()) / 86400000);
  if (days <= 0) return "Today";
  if (days === 1) return "Yesterday";
  if (days < 30) return `${days}d ago`;
  return formatDate(iso);
}

export function initials(name?: string) {
  if (!name) return "XH";
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]!.toUpperCase())
    .join("");
}
