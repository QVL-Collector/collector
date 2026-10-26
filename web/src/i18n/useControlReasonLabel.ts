import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { ControlReason } from '../shared/controlReason';

export function useControlReasonLabel(): (reason: ControlReason) => string {
  const { t } = useTranslation();
  return useCallback((reason: ControlReason) => t(`control.reasons.${reason}`), [t]);
}
