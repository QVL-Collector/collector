import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';

type AmountRowProps = {
  readonly label: string;
  readonly amount: string;
  readonly emphasized?: boolean;
};

export function AmountRow({ label, amount, emphasized = false }: AmountRowProps) {
  return (
    <Stack
      direction="row"
      spacing={2}
      justifyContent="space-between"
      alignItems="baseline"
      paddingTop={emphasized ? 2 : 0}
      sx={emphasized ? { borderTop: '0.0625rem solid', borderColor: 'divider' } : undefined}
    >
      <Typography
        component="dt"
        variant={emphasized ? 'subtitle1' : 'body1'}
        color={emphasized ? 'text.primary' : 'text.secondary'}
      >
        {label}
      </Typography>
      <Typography
        component="dd"
        margin={0}
        variant={emphasized ? 'h2' : 'body1'}
        color={emphasized ? 'primary' : 'text.primary'}
        fontWeight={emphasized ? 700 : 500}
        sx={{ fontVariantNumeric: 'tabular-nums', whiteSpace: 'nowrap' }}
      >
        {amount}
      </Typography>
    </Stack>
  );
}
