import { usePhotosClient } from '../api/usePhotosClient';
import { defaultPhotoLimits, type PhotoLimits } from '../domain/articlePhoto';

export function usePhotoLimits(): PhotoLimits {
  return usePhotosClient()?.limits ?? defaultPhotoLimits;
}
