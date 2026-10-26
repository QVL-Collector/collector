import { isRecord, isUnknownArray } from '../shared/payload';

export type FieldProblem = {
  readonly field: string;
  readonly code: string;
};

export type ProblemDetails = {
  readonly type: string | null;
  readonly fieldProblems: readonly FieldProblem[];
};

export const noProblemDetails: ProblemDetails = { type: null, fieldProblems: [] };

function toFieldProblems(value: unknown): readonly FieldProblem[] {
  if (!isRecord(value) || typeof value.champ !== 'string' || typeof value.code !== 'string') {
    return [];
  }
  return [{ field: value.champ, code: value.code }];
}

export function parseProblemDetails(payload: unknown): ProblemDetails {
  if (!isRecord(payload)) {
    return noProblemDetails;
  }
  const fieldProblems = isUnknownArray(payload.erreurs) ? payload.erreurs : [];
  return {
    type: typeof payload.type === 'string' ? payload.type : null,
    fieldProblems: fieldProblems.flatMap(toFieldProblems),
  };
}
