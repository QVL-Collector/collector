import type { RouteObject } from 'react-router';
import { CataloguePage } from '../features/catalogue/components/CataloguePage';
import { AppLayout } from './layout/AppLayout';
import { NotFoundPage } from './pages/NotFoundPage';
import { RouteErrorPage } from './pages/RouteErrorPage';

export const appRoutes: RouteObject[] = [
  {
    element: <AppLayout />,
    errorElement: <RouteErrorPage />,
    children: [
      { index: true, element: <CataloguePage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
];
