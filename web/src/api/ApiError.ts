export type ApiErrorKind = 'network' | 'http';

export type ProblemSummary = {
  readonly type: string | null;
  readonly title: string | null;
};

export class ApiError extends Error {
  readonly kind: ApiErrorKind;
  readonly status: number | null;
  readonly problem: ProblemSummary | null;

  constructor(kind: ApiErrorKind, status: number | null, problem: ProblemSummary | null) {
    super(status === null ? `API ${kind} error` : `API ${kind} error ${String(status)}`);
    this.name = 'ApiError';
    this.kind = kind;
    this.status = status;
    this.problem = problem;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
