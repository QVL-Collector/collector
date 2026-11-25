import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useCallback, useState } from 'react';
import { photosQueryKeys } from '../api/photosQueryKeys';
import { usePhotosClient } from '../api/usePhotosClient';
import type { ArticlePhoto } from '../domain/articlePhoto';
import { toPhotoFailure, type PhotoFailure } from './photoFailure';

export type PhotoRemovalModel = {
  readonly removingPhotoId: string | null;
  readonly failure: PhotoFailure | null;
  readonly removePhoto: (photo: ArticlePhoto) => void;
};

function withoutPhoto(photoId: string) {
  return (current: readonly ArticlePhoto[] | undefined): readonly ArticlePhoto[] =>
    (current ?? []).filter(({ id }) => id !== photoId);
}

export function usePhotoRemoval(articleId: string, onRemoved: () => void): PhotoRemovalModel {
  const client = usePhotosClient();
  const queryClient = useQueryClient();
  const [failure, setFailure] = useState<PhotoFailure | null>(null);
  const mutation = useMutation({
    mutationFn: async (photoId: string) => {
      if (client !== null) {
        await client.deletePhoto(articleId, photoId);
      }
    },
    onSuccess: (_, photoId) => {
      queryClient.setQueryData(photosQueryKeys.sellerArticle(articleId), withoutPhoto(photoId));
      onRemoved();
    },
    onError: (error) => {
      setFailure(toPhotoFailure(error));
    },
  });
  const { isPending, variables, mutate } = mutation;

  const removePhoto = useCallback(
    (photo: ArticlePhoto) => {
      if (isPending) {
        return;
      }
      setFailure(null);
      mutate(photo.id);
    },
    [isPending, mutate],
  );

  return {
    removingPhotoId: isPending ? variables : null,
    failure,
    removePhoto,
  };
}
