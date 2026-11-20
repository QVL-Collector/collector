import { useEffect, useState } from 'react';

export function useNow(intervalMs: number, enabled: boolean): Date {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    if (!enabled) {
      return;
    }
    const tick = () => {
      setNow(new Date());
    };
    tick();
    const timer = window.setInterval(tick, intervalMs);
    return () => {
      window.clearInterval(timer);
    };
  }, [intervalMs, enabled]);
  return now;
}
