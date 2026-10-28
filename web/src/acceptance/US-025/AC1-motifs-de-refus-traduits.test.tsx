import { act, renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import contractSource from '../../../../contracts/messages/controle.resultat/v1.schema.json?raw';
import { useArticleReasonLabel } from '../../i18n/useArticleReasonLabel';
import type { SupportedLanguage } from '../../i18n/languages';
import { controlReasons, type ControlReason } from '../../shared/controlReason';
import { switchLanguage } from '../US-010/catalogueHarness';

type ResultContract = { readonly $defs: { readonly motif: { readonly enum: readonly string[] } } };

const contractReasons = (JSON.parse(contractSource) as ResultContract).$defs.motif.enum;

async function labelsIn(language: SupportedLanguage): Promise<Map<string, string>> {
  await act(() => switchLanguage(language));
  const { result } = renderHook(() => useArticleReasonLabel());
  return new Map(
    contractReasons.map((reason) => [reason, result.current(reason as ControlReason)] as const),
  );
}

describe('US-025-AC1 motifs de refus du contrat controle.resultat v1', () => {
  it('Given le contrat, When ses motifs sont lus, Then la liste n’est pas vide', () => {
    expect(contractReasons.length).toBeGreaterThan(0);
  });

  it('Given le contrat, When il est comparé aux motifs connus du site, Then chaque motif du contrat est connu et réciproquement', () => {
    expect([...controlReasons].sort()).toEqual([...contractReasons].sort());
  });

  it.each(['fr', 'en'] as const)(
    'Given la langue %s, When chaque motif du contrat est libellé, Then le libellé est un texte non vide et jamais la clé brute',
    async (language) => {
      const labels = await labelsIn(language);

      for (const reason of contractReasons) {
        const label = labels.get(reason) ?? '';
        expect(label.trim(), reason).not.toBe('');
        expect(label, reason).not.toContain(reason);
        expect(label, reason).not.toContain('control.reasons');
      }
    },
  );

  it('Given les libellés FR et EN, When ils sont comparés, Then chaque motif est réellement traduit en anglais', async () => {
    const french = await labelsIn('fr');
    const english = await labelsIn('en');

    for (const reason of contractReasons) {
      expect(english.get(reason), reason).not.toBe(french.get(reason));
    }
  });

  it('Given des libellés en français, When l’utilisateur passe à l’anglais, Then le libellé affiché suit sans recréer le composant', async () => {
    await act(() => switchLanguage('fr'));
    const { result } = renderHook(() => useArticleReasonLabel());
    const french = result.current('annonce_incomplete');

    await act(() => switchLanguage('en'));

    expect(result.current('annonce_incomplete')).not.toBe(french);
    expect(result.current('annonce_incomplete').trim()).not.toBe('');
  });
});
