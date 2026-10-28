export class InvalidPayloadError extends Error {
  readonly path: string;

  constructor(path: string) {
    super(`Invalid payload at ${path}`);
    this.name = 'InvalidPayloadError';
    this.path = path;
  }
}

export type PayloadRecord = Readonly<Record<string, unknown>>;

export function isRecord(value: unknown): value is PayloadRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function isUnknownArray(value: unknown): value is readonly unknown[] {
  return Array.isArray(value);
}

export function expectRecord(value: unknown, path: string): PayloadRecord {
  if (!isRecord(value)) {
    throw new InvalidPayloadError(path);
  }
  return value;
}

export function expectArray(value: unknown, path: string): readonly unknown[] {
  if (!Array.isArray(value)) {
    throw new InvalidPayloadError(path);
  }
  return value;
}

export function expectString(value: unknown, path: string): string {
  if (typeof value !== 'string') {
    throw new InvalidPayloadError(path);
  }
  return value;
}

export function expectNonEmptyString(value: unknown, path: string): string {
  const text = expectString(value, path);
  if (text.trim() === '') {
    throw new InvalidPayloadError(path);
  }
  return text;
}

export function expectInteger(value: unknown, path: string, minimum: number): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < minimum) {
    throw new InvalidPayloadError(path);
  }
  return value;
}

export function expectOneOf<T extends string>(
  value: unknown,
  allowed: readonly T[],
  path: string,
): T {
  const match = allowed.find((candidate) => candidate === value);
  if (match === undefined) {
    throw new InvalidPayloadError(path);
  }
  return match;
}
