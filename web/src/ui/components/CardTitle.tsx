import Typography from '@mui/material/Typography';

type CardTitleProps = {
  readonly level: 'h2' | 'h3';
  readonly children: string;
};

export function CardTitle({ level, children }: CardTitleProps) {
  return (
    <Typography
      component={level}
      variant="h6"
      sx={{
        display: '-webkit-box',
        WebkitLineClamp: 3,
        WebkitBoxOrient: 'vertical',
        overflow: 'hidden',
        overflowWrap: 'anywhere',
      }}
    >
      {children}
    </Typography>
  );
}
