export const ordersQueryKeys = {
  detail: (id: string) => ['orders', 'detail', id] as const,
};
