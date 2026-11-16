import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

export type LockReason = 'not-modifiable' | 'not-submittable' | 'reserved';

export type LockNoticeHandlers = {
  readonly show: (reason: LockReason) => void;
  readonly clear: () => void;
};

export type LockNotice = {
  readonly message: string | null;
  readonly handlers: LockNoticeHandlers;
};

const messageKeys = {
  'not-modifiable': 'seller.form.locked',
  'not-submittable': 'seller.submission.locked',
  reserved: 'seller.reserved',
} as const satisfies Readonly<Record<LockReason, string>>;

export function useLockNotice(): LockNotice {
  const { t } = useTranslation();
  const [reason, setReason] = useState<LockReason | null>(null);
  const handlers = useMemo<LockNoticeHandlers>(
    () => ({
      show: setReason,
      clear: () => {
        setReason(null);
      },
    }),
    [],
  );
  return { message: reason === null ? null : t(messageKeys[reason]), handlers };
}
