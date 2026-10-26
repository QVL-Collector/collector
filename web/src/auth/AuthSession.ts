export type AuthSession = {
  readonly isAuthenticated: boolean;
  readonly getAccessToken: () => string | null;
};

export const anonymousSession: AuthSession = {
  isAuthenticated: false,
  getAccessToken: () => null,
};
