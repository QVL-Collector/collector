import { StrictMode } from 'react';
import type { Root } from 'react-dom/client';
import { loadRuntimeConfig } from '../config/loadRuntimeConfig';
import { initI18n } from '../i18n/initI18n';
import { detectInitialLanguage } from '../i18n/languages';
import { App } from './App';
import { AppThemeRoot } from './AppThemeRoot';
import { BootstrapFailure } from './BootstrapFailure';
import { createAppServices } from './createAppServices';

function renderFailure(root: Root): void {
  root.render(
    <StrictMode>
      <AppThemeRoot>
        <BootstrapFailure />
      </AppThemeRoot>
    </StrictMode>,
  );
}

export async function startApplication(root: Root): Promise<void> {
  await initI18n(detectInitialLanguage());
  try {
    const config = await loadRuntimeConfig();
    const services = await createAppServices(config);
    root.render(
      <StrictMode>
        <App services={services} />
      </StrictMode>,
    );
  } catch {
    renderFailure(root);
  }
}
