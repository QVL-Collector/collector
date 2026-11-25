import { useCallback, useState } from 'react';

export type ImageRevealModel = {
  readonly isLoaded: boolean;
  readonly previousSrc: string | null;
  readonly handleLoad: () => void;
};

export function useImageReveal(src: string, onLoad?: () => void): ImageRevealModel {
  const [loadedSrc, setLoadedSrc] = useState<string | null>(null);
  const handleLoad = useCallback(() => {
    setLoadedSrc(src);
    onLoad?.();
  }, [onLoad, src]);
  const isLoaded = loadedSrc === src;
  return { isLoaded, previousSrc: isLoaded ? null : loadedSrc, handleLoad };
}
