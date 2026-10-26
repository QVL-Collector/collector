export class ApiError extends Error {
  readonly status: number | null;

  constructor(status: number | null) {
    super(status === null ? 'API network error' : `API error ${String(status)}`);
    this.name = 'ApiError';
    this.status = status;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
