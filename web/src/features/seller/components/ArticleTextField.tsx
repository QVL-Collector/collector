import TextField from '@mui/material/TextField';
import type { ArticleFieldModel } from '../hooks/useArticleFieldModels';

type ArticleTextFieldProps = {
  readonly field: ArticleFieldModel;
  readonly label: string;
  readonly hint: string;
  readonly maxLength?: number;
  readonly multiline?: boolean;
  readonly inputMode?: 'text' | 'decimal';
};

export function ArticleTextField({
  field,
  label,
  hint,
  maxLength,
  multiline = false,
  inputMode = 'text',
}: ArticleTextFieldProps) {
  return (
    <TextField
      id={field.id}
      label={label}
      value={field.value}
      onChange={(event) => {
        field.onChange(event.target.value);
      }}
      inputRef={field.inputRef}
      error={field.error !== null}
      helperText={field.error ?? hint}
      multiline={multiline}
      minRows={multiline ? 5 : undefined}
      fullWidth
      slotProps={{ htmlInput: { maxLength, inputMode } }}
    />
  );
}
