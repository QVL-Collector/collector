import { isRecord } from '../shared/payload';

export const preserveFocusState = { preserveFocus: true } as const;

export function isFocusPreserved(state: unknown): boolean {
  return isRecord(state) && state.preserveFocus === true;
}
