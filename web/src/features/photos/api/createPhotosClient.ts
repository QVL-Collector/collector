import type { HttpClient } from '../../../api/httpClient';
import type { CatalogueConfig } from '../../../config/runtimeConfig';
import type { PhotoLimits, PhotosClient } from '../domain/articlePhoto';
import { createHttpPhotosClient } from './httpPhotosClient';

export async function createPhotosClient(
  config: CatalogueConfig,
  http: HttpClient,
  limits: PhotoLimits,
): Promise<PhotosClient> {
  switch (config.source) {
    case 'api':
      return createHttpPhotosClient(http, limits);
    case 'demo': {
      const { createDemoPhotosClient } = await import('./demoPhotosClient');
      return createDemoPhotosClient(config.demoScenario, limits);
    }
  }
}
