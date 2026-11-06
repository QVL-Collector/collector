import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormHelperText from '@mui/material/FormHelperText';
import FormLabel from '@mui/material/FormLabel';
import Radio from '@mui/material/Radio';
import RadioGroup from '@mui/material/RadioGroup';
import Stack from '@mui/material/Stack';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import type { RefusalDialogModel } from '../hooks/useRefusalDialog';

type RefusalDialogProps = {
  readonly title: string;
  readonly model: RefusalDialogModel;
};

export function RefusalDialog({ title, model }: RefusalDialogProps) {
  const { t } = useTranslation();
  const titleId = useId();
  const descriptionId = useId();
  const legendId = useId();
  const helperId = useId();
  return (
    <Dialog
      open={model.open}
      onClose={model.close}
      aria-labelledby={titleId}
      aria-describedby={descriptionId}
      fullWidth
      maxWidth="sm"
    >
      <DialogTitle id={titleId}>{t('admin.review.refusal.title')}</DialogTitle>
      <DialogContent>
        <Stack spacing={2.5}>
          <DialogContentText id={descriptionId} sx={{ overflowWrap: 'anywhere' }}>
            {t('admin.review.refusal.description', { title })}
          </DialogContentText>
          <FormControl component="fieldset" error={model.missingReason} required>
            <FormLabel id={legendId} component="legend">
              {t('admin.review.refusal.reasonLabel')}
            </FormLabel>
            <RadioGroup
              aria-labelledby={legendId}
              aria-describedby={model.missingReason ? helperId : undefined}
              name={legendId}
              value={model.reason}
              onChange={model.changeReason}
            >
              {model.options.map((option) => (
                <FormControlLabel
                  key={option.value}
                  value={option.value}
                  control={<Radio />}
                  label={option.label}
                />
              ))}
            </RadioGroup>
            {model.missingReason && (
              <FormHelperText id={helperId} role="alert">
                {t('admin.review.refusal.missingReason')}
              </FormHelperText>
            )}
          </FormControl>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={model.close}>{t('admin.review.refusal.cancel')}</Button>
        <Button variant="contained" color="error" onClick={model.confirm}>
          {t('admin.review.refusal.confirm')}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
