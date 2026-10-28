import { useEffect, useRef, type RefObject } from 'react';
import { useLocation, type Location } from 'react-router';
import { useDocumentTitle } from '../ui/hooks/useDocumentTitle';
import { isFocusPreserved } from './focusPreservation';

const initialLocationKey = 'default';

export function usePageIdentity(documentTitle: string): RefObject<HTMLHeadingElement | null> {
  const headingRef = useRef<HTMLHeadingElement>(null);
  const location: Location<unknown> = useLocation();
  const { key } = location;
  const preserveFocus = isFocusPreserved(location.state);
  useDocumentTitle(documentTitle);
  useEffect(() => {
    if (key !== initialLocationKey && !preserveFocus) {
      headingRef.current?.focus();
    }
  }, [key, preserveFocus]);
  return headingRef;
}
