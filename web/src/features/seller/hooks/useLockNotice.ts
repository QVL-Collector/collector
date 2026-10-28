import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

export type LockReason = 'not-modifiable' | 'not-submittable';

export type LockNoticeHandlers = {
  readonly show: (reason: LockReason) => void;
  readonly clear: () => void;
};

export type LockNotice = {
  readonly message: string | null;
  readonly handlers: LockNoticeHandlers;
};

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
  const messageOf = (locked: LockReason) =>
    locked === 'not-modifiable' ? t('seller.form.locked') : t('seller.submission.locked');
  return { message: reason === null ? null : messageOf(reason), handlers };
}
