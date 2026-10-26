import { useCallback, type MouseEvent } from 'react';

export type SelectableOption<T extends string> = {
  readonly value: T;
};

export function useExclusiveSelection<T extends string>(
  options: readonly SelectableOption<T>[],
  onSelect: (value: T) => void,
): (event: MouseEvent<HTMLElement>, next: unknown) => void {
  return useCallback(
    (_event: MouseEvent<HTMLElement>, next: unknown) => {
      const selected = options.find((option) => option.value === next);
      if (selected !== undefined) {
        onSelect(selected.value);
      }
    },
    [options, onSelect],
  );
}
