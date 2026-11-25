import UploadFileRounded from '@mui/icons-material/UploadFileRounded';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import type { SxProps, Theme } from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import { useId, type ReactNode } from 'react';
import { springEasing } from '../animations/springEasing';
import { useFileDrop } from '../hooks/useFileDrop';

type FileDropZoneProps = {
  readonly title: string;
  readonly description: string;
  readonly actionLabel: string;
  readonly accept: string;
  readonly icon: ReactNode;
  readonly disabled?: boolean;
  readonly onFiles: (files: readonly File[]) => void;
};

const springDurationMs = 420;

function dropZoneSx(isDragActive: boolean): SxProps<Theme> {
  return (theme) => ({
    borderRadius: '1rem',
    border: `0.125rem dashed ${isDragActive ? theme.palette.primary.main : theme.palette.divider}`,
    backgroundColor: isDragActive ? theme.palette.action.hover : theme.palette.background.paper,
    transition: theme.transitions.create(['border-color', 'background-color']),
  });
}

function dropIconSx(isDragActive: boolean): SxProps<Theme> {
  return (theme) => ({
    width: '3.5rem',
    height: '3.5rem',
    borderRadius: '50%',
    fontSize: '1.75rem',
    color: theme.palette.primary.contrastText,
    backgroundColor: theme.palette.primary.main,
    transform: isDragActive ? 'translateY(-0.25rem) scale(1.08)' : 'none',
    transition: theme.transitions.create('transform', {
      duration: springDurationMs,
      easing: springEasing,
    }),
    '@media (prefers-reduced-motion: reduce)': { transform: 'none' },
  });
}

export function FileDropZone({
  title,
  description,
  actionLabel,
  accept,
  icon,
  disabled = false,
  onFiles,
}: FileDropZoneProps) {
  const descriptionId = useId();
  const { inputRef, isDragActive, dropHandlers, openPicker, onInputChange } = useFileDrop(
    onFiles,
    disabled,
  );
  return (
    <Stack
      {...dropHandlers}
      alignItems="center"
      spacing={1.5}
      paddingX={{ xs: 2, md: 4 }}
      paddingY={{ xs: 3, md: 4 }}
      textAlign="center"
      sx={dropZoneSx(isDragActive)}
    >
      <Stack alignItems="center" justifyContent="center" aria-hidden sx={dropIconSx(isDragActive)}>
        {icon}
      </Stack>
      <Typography component="p" variant="subtitle1">
        {title}
      </Typography>
      <Typography id={descriptionId} variant="body2" color="text.secondary" maxWidth="28rem">
        {description}
      </Typography>
      <Button
        variant="contained"
        startIcon={<UploadFileRounded />}
        disabled={disabled}
        onClick={openPicker}
        aria-describedby={descriptionId}
        sx={{ minHeight: '2.75rem' }}
      >
        {actionLabel}
      </Button>
      <input
        ref={inputRef}
        type="file"
        accept={accept}
        hidden
        tabIndex={-1}
        aria-label={actionLabel}
        disabled={disabled}
        onChange={onInputChange}
      />
    </Stack>
  );
}
