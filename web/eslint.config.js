import js from '@eslint/js';
import prettier from 'eslint-config-prettier';
import i18next from 'eslint-plugin-i18next';
import jsxA11y from 'eslint-plugin-jsx-a11y';
import noUnsanitized from 'eslint-plugin-no-unsanitized';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import globals from 'globals';
import tseslint from 'typescript-eslint';

const storageMessage = 'Jetons et données de session en mémoire uniquement (D06).';
const literalTextMessage =
  'Texte visible ou lu par les aides techniques : passer par t() (ENF-30).';
const accessibleTextAttributes = [
  'aria-label',
  'aria-description',
  'aria-placeholder',
  'aria-roledescription',
  'aria-valuetext',
  'alt',
  'title',
  'placeholder',
  'label',
  'helperText',
  'subtitle',
  'description',
  'message',
].join('|');
const textAttribute = `JSXAttribute[name.name=/^(${accessibleTextAttributes})$/]`;
const literalText = ':matches(Literal, TemplateLiteral[expressions.length=0])';
const literalTextSelectors = [
  `${textAttribute} > ${literalText}`,
  `${textAttribute} > JSXExpressionContainer > ${literalText}`,
].map((selector) => ({ selector, message: literalTextMessage }));

export default tseslint.config(
  { ignores: ['dist', 'coverage', 'test-results', 'node_modules'] },
  js.configs.recommended,
  ...tseslint.configs.strictTypeChecked,
  ...tseslint.configs.stylisticTypeChecked,
  {
    languageOptions: {
      globals: globals.browser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
  },
  react.configs.flat.recommended,
  react.configs.flat['jsx-runtime'],
  jsxA11y.flatConfigs.strict,
  noUnsanitized.configs.recommended,
  {
    files: ['src/**/*.tsx'],
    ...i18next.configs['flat/recommended'],
  },
  {
    files: ['src/**/*.tsx'],
    rules: {
      'no-restricted-syntax': ['error', ...literalTextSelectors],
    },
  },
  {
    plugins: { 'react-hooks': reactHooks },
    settings: { react: { version: 'detect' } },
    rules: {
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'error',
      'react/no-danger': 'error',
      'react/no-danger-with-children': 'error',
      'react/jsx-no-script-url': 'error',
      'react/jsx-no-target-blank': 'error',
      'react/prop-types': 'off',
      'no-eval': 'error',
      'no-implied-eval': 'error',
      'no-new-func': 'error',
      'no-script-url': 'error',
      'no-restricted-globals': [
        'error',
        { name: 'localStorage', message: storageMessage },
        { name: 'sessionStorage', message: storageMessage },
      ],
      'no-restricted-properties': [
        'error',
        { object: 'window', property: 'localStorage', message: storageMessage },
        { object: 'window', property: 'sessionStorage', message: storageMessage },
        { object: 'document', property: 'cookie', message: storageMessage },
      ],
      '@typescript-eslint/consistent-type-definitions': ['error', 'type'],
      '@typescript-eslint/switch-exhaustiveness-check': 'error',
      '@typescript-eslint/consistent-type-imports': 'error',
    },
  },
  {
    files: ['src/auth/createOidcStateStore.ts'],
    rules: {
      'no-restricted-properties': [
        'error',
        { object: 'window', property: 'localStorage', message: storageMessage },
        { object: 'document', property: 'cookie', message: storageMessage },
      ],
    },
  },
  {
    files: ['**/*.d.ts'],
    rules: { '@typescript-eslint/consistent-type-definitions': 'off' },
  },
  {
    files: ['**/*.js'],
    ...tseslint.configs.disableTypeChecked,
  },
  prettier,
);
