interface PageSectionHeaderProps {
  title: string;
  subtitle: string;
}

export function PageSectionHeader({ title, subtitle }: PageSectionHeaderProps) {
  return (
    <div className="page-header">
      <h1>{title}</h1>
      <p>{subtitle}</p>
    </div>
  );
}
