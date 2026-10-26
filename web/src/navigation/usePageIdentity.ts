import { useEffect, useRef, type RefObject } from 'react';
import { useLocation } from 'react-router';
import { useDocumentTitle } from '../ui/hooks/useDocumentTitle';

const initialLocationKey = 'default';

export function usePageIdentity(documentTitle: string): RefObject<HTMLHeadingElement | null> {
  const headingRef = useRef<HTMLHeadingElement>(null);
  const { key } = useLocation();
  useDocumentTitle(documentTitle);
  useEffect(() => {
    if (key !== initialLocationKey) {
      headingRef.current?.focus();
    }
  }, [key]);
  return headingRef;
}
