import BrokenImageRounded from '@mui/icons-material/BrokenImageRounded';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import type { SxProps, Theme } from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import { springEasing } from '../animations/springEasing';
import { useImageReveal } from '../hooks/useImageReveal';

export type ImageFit = 'cover' | 'contain';

type ImageFrameProps = {
  readonly src: string;
  readonly alt: string;
  readonly fit: ImageFit;
  readonly isBroken: boolean;
  readonly unavailableLabel: string;
  readonly loading?: 'lazy' | 'eager';
  readonly onError?: () => void;
  readonly onLoad?: () => void;
};

const revealDurationMs = 480;

const frameSx: SxProps<Theme> = (theme) => ({
  aspectRatio: '4 / 3',
  borderRadius: '1rem',
  backgroundColor: theme.palette.action.hover,
  '&:hover img': { transform: 'scale(1.03)' },
  '@media (prefers-reduced-motion: reduce)': { '&:hover img': { transform: 'none' } },
});

const layer = {
  position: 'absolute',
  inset: 0,
  display: 'block',
  width: '100%',
  height: '100%',
} as const;

function layerSx(fit: ImageFit): SxProps<Theme> {
  return { ...layer, objectFit: fit };
}

function imageSx(isLoaded: boolean, fit: ImageFit): SxProps<Theme> {
  return (theme) => ({
    ...layer,
    objectFit: fit,
    opacity: isLoaded ? 1 : 0,
    transform: isLoaded ? 'none' : 'scale(1.02)',
    transition: theme.transitions.create(['opacity', 'transform'], {
      duration: revealDurationMs,
      easing: springEasing,
    }),
  });
}

export function ImageFrame({
  src,
  alt,
  fit,
  isBroken,
  unavailableLabel,
  loading = 'lazy',
  onError,
  onLoad,
}: ImageFrameProps) {
  const { isLoaded, previousSrc, handleLoad } = useImageReveal(src, onLoad);
  return (
    <Box position="relative" overflow="hidden" sx={frameSx}>
      {isBroken ? (
        <Stack
          height="100%"
          alignItems="center"
          justifyContent="center"
          spacing={1}
          padding={2}
          color="text.secondary"
        >
          <BrokenImageRounded aria-hidden />
          <Typography variant="body2" textAlign="center">
            {unavailableLabel}
          </Typography>
        </Stack>
      ) : (
        [
          previousSrc === null ? null : (
            <Box
              key={previousSrc}
              component="img"
              src={previousSrc}
              alt={alt}
              aria-hidden
              referrerPolicy="no-referrer"
              sx={layerSx(fit)}
            />
          ),
          <Box
            key={src}
            component="img"
            src={src}
            alt={alt}
            loading={loading}
            decoding="async"
            referrerPolicy="no-referrer"
            onError={onError}
            onLoad={handleLoad}
            sx={imageSx(isLoaded, fit)}
          />,
        ]
      )}
    </Box>
  );
}
