import Alert from '@mui/material/Alert';
import Snackbar from '@mui/material/Snackbar';

type ErrorToastProps = {
  readonly open: boolean;
  readonly message: string;
  readonly onClose: () => void;
};

const autoHideDurationMs = 8000;

export function ErrorToast({ open, message, onClose }: ErrorToastProps) {
  return (
    <Snackbar
      open={open}
      autoHideDuration={autoHideDurationMs}
      onClose={onClose}
      anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
    >
      <Alert severity="error" variant="filled" onClose={onClose} sx={{ width: '100%' }}>
        {message}
      </Alert>
    </Snackbar>
  );
}
