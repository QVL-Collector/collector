import type { RouteObject } from 'react-router';
import { signInCallbackPath } from '../auth/authPaths';
import { CataloguePage } from '../features/catalogue/components/CataloguePage';
import { SellerSpacePage } from '../features/seller/components/SellerSpacePage';
import { sellerSpacePath } from '../features/seller/sellerPaths';
import { AppLayout } from './layout/AppLayout';
import { NotFoundPage } from './pages/NotFoundPage';
import { RouteErrorPage } from './pages/RouteErrorPage';
import { SignInCallbackPage } from './pages/SignInCallbackPage';
import { RequireAuthentication } from './routing/RequireAuthentication';

export const appRoutes: RouteObject[] = [
  {
    element: <AppLayout />,
    errorElement: <RouteErrorPage />,
    children: [
      { index: true, element: <CataloguePage /> },
      { path: signInCallbackPath, element: <SignInCallbackPage /> },
      {
        element: <RequireAuthentication requiredRole="vendeur" />,
        children: [{ path: sellerSpacePath, element: <SellerSpacePage /> }],
      },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
];
