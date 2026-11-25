import { useEffect, useState } from 'react';

export function useObjectUrl(blob: Blob | null): string | null {
  const [url, setUrl] = useState<string | null>(null);
  useEffect(() => {
    if (blob === null) {
      setUrl(null);
      return undefined;
    }
    const created = URL.createObjectURL(blob);
    setUrl(created);
    return () => {
      URL.revokeObjectURL(created);
    };
  }, [blob]);
  return blob === null ? null : url;
}
