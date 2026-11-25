import {
  useCallback,
  useRef,
  useState,
  type ChangeEvent,
  type DragEvent,
  type RefObject,
} from 'react';

export type FileDropHandlers = {
  readonly onDragEnter: (event: DragEvent<HTMLElement>) => void;
  readonly onDragOver: (event: DragEvent<HTMLElement>) => void;
  readonly onDragLeave: (event: DragEvent<HTMLElement>) => void;
  readonly onDrop: (event: DragEvent<HTMLElement>) => void;
};

export type FileDropModel = {
  readonly inputRef: RefObject<HTMLInputElement | null>;
  readonly isDragActive: boolean;
  readonly dropHandlers: FileDropHandlers;
  readonly openPicker: () => void;
  readonly onInputChange: (event: ChangeEvent<HTMLInputElement>) => void;
};

function carriesFiles(event: DragEvent<HTMLElement>): boolean {
  return event.dataTransfer.types.includes('Files');
}

export function useFileDrop(
  onFiles: (files: readonly File[]) => void,
  disabled: boolean,
): FileDropModel {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const dragDepth = useRef(0);
  const [isDragActive, setIsDragActive] = useState(false);

  const resetDrag = useCallback(() => {
    dragDepth.current = 0;
    setIsDragActive(false);
  }, []);

  const onDragEnter = useCallback(
    (event: DragEvent<HTMLElement>) => {
      if (!carriesFiles(event)) {
        return;
      }
      event.preventDefault();
      dragDepth.current += 1;
      setIsDragActive(!disabled);
    },
    [disabled],
  );

  const onDragOver = useCallback(
    (event: DragEvent<HTMLElement>) => {
      if (!carriesFiles(event)) {
        return;
      }
      event.preventDefault();
      event.dataTransfer.dropEffect = disabled ? 'none' : 'copy';
    },
    [disabled],
  );

  const onDragLeave = useCallback(() => {
    dragDepth.current = Math.max(dragDepth.current - 1, 0);
    if (dragDepth.current === 0) {
      setIsDragActive(false);
    }
  }, []);

  const onDrop = useCallback(
    (event: DragEvent<HTMLElement>) => {
      event.preventDefault();
      resetDrag();
      if (!disabled) {
        onFiles([...event.dataTransfer.files]);
      }
    },
    [disabled, onFiles, resetDrag],
  );

  const openPicker = useCallback(() => {
    inputRef.current?.click();
  }, []);

  const onInputChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const files = [...(event.target.files ?? [])];
      event.target.value = '';
      if (files.length > 0) {
        onFiles(files);
      }
    },
    [onFiles],
  );

  return {
    inputRef,
    isDragActive,
    dropHandlers: { onDragEnter, onDragOver, onDragLeave, onDrop },
    openPicker,
    onInputChange,
  };
}
