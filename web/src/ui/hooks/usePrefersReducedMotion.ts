import useMediaQuery from '@mui/material/useMediaQuery';

const reducedMotionQuery = '(prefers-reduced-motion: reduce)';

export function usePrefersReducedMotion(): boolean {
  return useMediaQuery(reducedMotionQuery, { noSsr: true });
}
