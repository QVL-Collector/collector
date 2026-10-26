import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import { useExclusiveSelection, type SelectableOption } from '../hooks/useExclusiveSelection';

export type LanguageOption<T extends string> = SelectableOption<T> & {
  readonly shortLabel: string;
  readonly fullLabel: string;
};

type LanguageSwitcherProps<T extends string> = {
  readonly label: string;
  readonly value: T;
  readonly options: readonly LanguageOption<T>[];
  readonly onChange: (value: T) => void;
};

export function LanguageSwitcher<T extends string>({
  label,
  value,
  options,
  onChange,
}: LanguageSwitcherProps<T>) {
  const handleChange = useExclusiveSelection(options, onChange);
  return (
    <ToggleButtonGroup
      exclusive
      size="small"
      color="primary"
      value={value}
      aria-label={label}
      onChange={handleChange}
    >
      {options.map((option) => (
        <ToggleButton
          key={option.value}
          value={option.value}
          lang={option.value}
          aria-label={option.fullLabel}
          sx={{ minWidth: '2.75rem', minHeight: '2.75rem', fontWeight: 500 }}
        >
          {option.shortLabel}
        </ToggleButton>
      ))}
    </ToggleButtonGroup>
  );
}
