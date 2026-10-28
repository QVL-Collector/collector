import { screen } from '@testing-library/react';
import type { SupportedLanguage } from '../../i18n/languages';

export type SellerWording = {
  readonly sellerSpace: string;
  readonly myItems: string;
  readonly newItem: string;
  readonly newItemTitle: string;
  readonly title: string;
  readonly description: string;
  readonly category: string;
  readonly price: string;
  readonly shippingCost: string;
  readonly create: string;
  readonly save: string;
  readonly submit: string;
  readonly reload: string;
  readonly accessDenied: string;
  readonly states: Readonly<
    Record<'brouillon' | 'en_controle' | 'publie' | 'refuse' | 'a_verifier', string>
  >;
  readonly incomplete: string;
  readonly unknownCategory: string;
};

export const sellerWording: Readonly<Record<SupportedLanguage, SellerWording>> = {
  fr: {
    sellerSpace: 'Espace vendeur',
    myItems: 'Mes articles',
    newItem: 'Nouvel article',
    newItemTitle: 'Nouvel article',
    title: 'Titre',
    description: 'Description',
    category: 'Catégorie',
    price: 'Prix (€)',
    shippingCost: 'Frais de port (€)',
    create: 'Créer le brouillon',
    save: 'Enregistrer le brouillon',
    submit: 'Soumettre au contrôle',
    reload: 'Recharger la dernière version',
    accessDenied: 'Accès réservé',
    states: {
      brouillon: 'Brouillon',
      en_controle: 'En contrôle',
      publie: 'Publié',
      refuse: 'Refusé',
      a_verifier: 'À vérifier',
    },
    incomplete: 'Annonce incomplète',
    unknownCategory: 'Catégorie inconnue',
  },
  en: {
    sellerSpace: 'Seller area',
    myItems: 'My items',
    newItem: 'New item',
    newItemTitle: 'New item',
    title: 'Title',
    description: 'Description',
    category: 'Category',
    price: 'Price (€)',
    shippingCost: 'Shipping cost (€)',
    create: 'Create the draft',
    save: 'Save the draft',
    submit: 'Submit for review',
    reload: 'Reload the latest version',
    accessDenied: 'Restricted access',
    states: {
      brouillon: 'Draft',
      en_controle: 'Under review',
      publie: 'Published',
      refuse: 'Rejected',
      a_verifier: 'To be checked',
    },
    incomplete: 'Incomplete listing',
    unknownCategory: 'Unknown category',
  },
};

export const fr = sellerWording.fr;

export function field(name: string): HTMLElement {
  return screen.getByRole('textbox', { name });
}

export function categoryField(name = fr.category): HTMLSelectElement {
  return screen.getByRole<HTMLSelectElement>('combobox', { name });
}

export async function editorHeading(title: string): Promise<HTMLElement> {
  return screen.findByRole('heading', { level: 1, name: title });
}
