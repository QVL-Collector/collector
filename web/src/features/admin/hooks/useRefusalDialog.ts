import { useCallback, useMemo, useState, type ChangeEvent } from 'react';
import { useArticleReasonLabel } from '../../../i18n/useArticleReasonLabel';
import { refusalReasons, type RefusalReason } from '../../../shared/articleReason';

export type RefusalReasonOption = {
  readonly value: RefusalReason;
  readonly label: string;
};

export type RefusalDialogModel = {
  readonly open: boolean;
  readonly reason: RefusalReason | '';
  readonly missingReason: boolean;
  readonly options: readonly RefusalReasonOption[];
  readonly openDialog: () => void;
  readonly close: () => void;
  readonly changeReason: (event: ChangeEvent<HTMLInputElement>, value: string) => void;
  readonly confirm: () => void;
};

function toRefusalReason(value: string): RefusalReason | null {
  return refusalReasons.find((reason) => reason === value) ?? null;
}

export function useRefusalDialog(onConfirm: (reason: RefusalReason) => void): RefusalDialogModel {
  const reasonLabel = useArticleReasonLabel();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState<RefusalReason | null>(null);
  const [missingReason, setMissingReason] = useState(false);

  const options = useMemo(
    () => refusalReasons.map((value) => ({ value, label: reasonLabel(value) })),
    [reasonLabel],
  );

  const openDialog = useCallback(() => {
    setReason(null);
    setMissingReason(false);
    setOpen(true);
  }, []);

  const close = useCallback(() => {
    setOpen(false);
  }, []);

  const changeReason = useCallback((_event: ChangeEvent<HTMLInputElement>, value: string) => {
    setReason(toRefusalReason(value));
    setMissingReason(false);
  }, []);

  const confirm = useCallback(() => {
    if (reason === null) {
      setMissingReason(true);
      return;
    }
    setOpen(false);
    onConfirm(reason);
  }, [reason, onConfirm]);

  return {
    open,
    reason: reason ?? '',
    missingReason,
    options,
    openDialog,
    close,
    changeReason,
    confirm,
  };
}
