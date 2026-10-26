import { noProblemDetails, type ProblemDetails } from './problemDetails';

export class ApiError extends Error {
  readonly status: number | null;
  readonly problem: ProblemDetails;

  constructor(status: number | null, problem: ProblemDetails = noProblemDetails) {
    super(status === null ? 'API network error' : `API error ${String(status)}`);
    this.name = 'ApiError';
    this.status = status;
    this.problem = problem;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
