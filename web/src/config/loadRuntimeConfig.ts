import { parseRuntimeConfig, type RuntimeConfig } from './runtimeConfig';

const runtimeConfigPath = '/config.json';

export async function loadRuntimeConfig(): Promise<RuntimeConfig> {
  const response = await fetch(runtimeConfigPath, {
    cache: 'no-store',
    credentials: 'omit',
    headers: { Accept: 'application/json' },
  });
  if (!response.ok) {
    throw new Error(`Runtime configuration unavailable (${String(response.status)})`);
  }
  const payload: unknown = await response.json();
  return parseRuntimeConfig(payload);
}
