import { toSupportedLanguage, type SupportedLanguage } from '../i18n/languages';
import { isRecord } from '../shared/payload';
import { toSafeReturnPath } from './returnPath';

export type SignInIntent = {
  readonly returnPath: string;
  readonly language: SupportedLanguage;
};

export type SignInOutcome =
  | {
      readonly status: 'completed';
      readonly returnPath: string;
      readonly language: SupportedLanguage | undefined;
    }
  | { readonly status: 'failed'; readonly language: SupportedLanguage | undefined };

export function toSignInState(intent: SignInIntent, origin: string): SignInIntent {
  return { returnPath: toSafeReturnPath(intent.returnPath, origin), language: intent.language };
}

export function readReturnPath(state: unknown, origin: string): string {
  return toSafeReturnPath(isRecord(state) ? state.returnPath : undefined, origin);
}

export function readLanguage(state: unknown): SupportedLanguage | undefined {
  const language = isRecord(state) ? state.language : undefined;
  return typeof language === 'string' ? toSupportedLanguage(language) : undefined;
}

export function readFailedSignInLanguage(error: unknown): SupportedLanguage | undefined {
  return readLanguage(isRecord(error) ? error.state : undefined);
}
