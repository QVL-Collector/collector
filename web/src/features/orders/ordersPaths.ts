export const buyerSpacePath = '/acheteur';

export const orderRoutePattern = `${buyerSpacePath}/commandes/:orderId`;

export function orderPath(id: string): string {
  return `${buyerSpacePath}/commandes/${encodeURIComponent(id)}`;
}
