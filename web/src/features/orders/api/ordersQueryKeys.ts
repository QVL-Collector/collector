export const ordersQueryKeys = {
  detail: (id: string) => ['orders', 'detail', id] as const,
  lists: () => ['orders', 'list'] as const,
  list: (page: number, pageSize: number) => ['orders', 'list', page, pageSize] as const,
};
