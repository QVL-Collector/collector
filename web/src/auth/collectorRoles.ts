export const collectorRoles = ['admin', 'vendeur', 'acheteur'] as const;

export type CollectorRole = (typeof collectorRoles)[number];

export function toCollectorRoles(values: readonly unknown[]): readonly CollectorRole[] {
  return collectorRoles.filter((role) => values.includes(role));
}

export function primaryRole(roles: readonly CollectorRole[]): CollectorRole | null {
  return collectorRoles.find((role) => roles.includes(role)) ?? null;
}
