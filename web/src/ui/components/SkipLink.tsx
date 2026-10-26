import Link from '@mui/material/Link';

type SkipLinkProps = {
  readonly targetId: string;
  readonly label: string;
};

export function SkipLink({ targetId, label }: SkipLinkProps) {
  return (
    <Link
      href={`#${targetId}`}
      variant="button"
      underline="always"
      sx={{
        position: 'absolute',
        top: '0.75rem',
        left: '0.75rem',
        zIndex: (theme) => theme.zIndex.tooltip,
        paddingX: 2,
        paddingY: 1,
        borderRadius: '0.5rem',
        backgroundColor: 'background.paper',
        boxShadow: 3,
        transform: 'translateY(-200%)',
        '&:focus-visible': { transform: 'none' },
      }}
    >
      {label}
    </Link>
  );
}
