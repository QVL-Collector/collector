const maxInitials = 2;

export function initialsOf(name: string): string {
  return name
    .trim()
    .split(/\s+/)
    .slice(0, maxInitials)
    .map((word) => word.charAt(0).toLocaleUpperCase())
    .join('');
}
