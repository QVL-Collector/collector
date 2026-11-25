import { keyframes, type SxProps, type Theme } from '@mui/material/styles';
import { springEasing } from './springEasing';

const rise = keyframes`
  from {
    opacity: 0;
    transform: translateY(0.75rem) scale(0.98);
  }
  to {
    opacity: 1;
    transform: none;
  }
`;

const durationMs = 520;
const staggerMs = 45;
const maxStaggeredItems = 12;

export function revealSx(index: number): SxProps<Theme> {
  const delayMs = Math.min(index, maxStaggeredItems) * staggerMs;
  return {
    animation: `${rise} ${String(durationMs)}ms ${springEasing} both`,
    animationDelay: `${String(delayMs)}ms`,
    '@media (prefers-reduced-motion: reduce)': { animation: 'none' },
  };
}
