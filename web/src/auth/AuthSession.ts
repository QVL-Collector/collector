export type AuthSession = {
  readonly getAccessToken: () => string | null;
};

export const anonymousSession: AuthSession = {
  getAccessToken: () => null,
};
