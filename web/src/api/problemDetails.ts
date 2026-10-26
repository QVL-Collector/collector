import { isRecord } from '../shared/payload';
import type { ProblemSummary } from './ApiError';

const problemContentType = 'application/problem+json';

function readOptionalString(value: unknown): string | null {
  return typeof value === 'string' ? value : null;
}

export async function readProblemSummary(response: Response): Promise<ProblemSummary | null> {
  const contentType = response.headers.get('Content-Type') ?? '';
  if (!contentType.startsWith(problemContentType)) {
    return null;
  }
  try {
    const payload: unknown = await response.json();
    if (!isRecord(payload)) {
      return null;
    }
    return { type: readOptionalString(payload.type), title: readOptionalString(payload.title) };
  } catch {
    return null;
  }
}
