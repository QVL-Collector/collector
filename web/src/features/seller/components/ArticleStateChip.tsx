import CheckCircleRounded from '@mui/icons-material/CheckCircleRounded';
import EditNoteRounded from '@mui/icons-material/EditNoteRounded';
import HighlightOffRounded from '@mui/icons-material/HighlightOffRounded';
import HourglassTopRounded from '@mui/icons-material/HourglassTopRounded';
import ManageSearchRounded from '@mui/icons-material/ManageSearchRounded';
import Chip, { type ChipProps } from '@mui/material/Chip';
import type { ReactElement } from 'react';
import { revealSx } from '../../../ui/animations/revealSx';
import type { ArticleState } from '../domain/sellerArticle';

type StateAppearance = {
  readonly icon: ReactElement;
  readonly color: ChipProps['color'];
  readonly variant: ChipProps['variant'];
};

function appearanceOf(state: ArticleState): StateAppearance {
  switch (state) {
    case 'brouillon':
      return { icon: <EditNoteRounded />, color: 'default', variant: 'outlined' };
    case 'en_controle':
      return { icon: <HourglassTopRounded />, color: 'primary', variant: 'outlined' };
    case 'publie':
      return { icon: <CheckCircleRounded />, color: 'primary', variant: 'filled' };
    case 'refuse':
      return { icon: <HighlightOffRounded />, color: 'error', variant: 'outlined' };
    case 'a_verifier':
      return { icon: <ManageSearchRounded />, color: 'secondary', variant: 'outlined' };
  }
}

type ArticleStateChipProps = {
  readonly state: ArticleState;
  readonly label: string;
};

export function ArticleStateChip({ state, label }: ArticleStateChipProps) {
  const { icon, color, variant } = appearanceOf(state);
  return (
    <Chip key={state} icon={icon} label={label} color={color} variant={variant} sx={revealSx(0)} />
  );
}
