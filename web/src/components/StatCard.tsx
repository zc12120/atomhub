interface StatCardProps {
  label: string;
  value: number;
}

const numberFormatter = new Intl.NumberFormat();

export default function StatCard({ label, value }: StatCardProps): JSX.Element {
  return (
    <article className="stat-card">
      <p className="stat-label">{label}</p>
      <strong className="stat-value">{numberFormatter.format(value)}</strong>
    </article>
  );
}
