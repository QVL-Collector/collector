import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useRef, useState } from 'react';
import { isAbortError } from '../../../api/httpClient';
import { useObjectUrl } from '../../../ui/hooks/useObjectUrl';
import { photosQueryKeys } from '../api/photosQueryKeys';
import { usePhotosClient } from '../api/usePhotosClient';
import type { ArticlePhoto } from '../domain/articlePhoto';
import { inspectPhotoFile } from '../domain/photoInspection';
import { toPhotoFailure, type PhotoFailure } from './photoFailure';

export type PhotoUploadPhase =
  | { readonly kind: 'idle' }
  | { readonly kind: 'checking' }
  | { readonly kind: 'uploading'; readonly progress: number }
  | { readonly kind: 'failed'; readonly failure: PhotoFailure };

export type PhotoUploadModel = {
  readonly phase: PhotoUploadPhase;
  readonly previewUrl: string | null;
  readonly isBusy: boolean;
  readonly selectFiles: (files: readonly File[]) => void;
  readonly cancel: () => void;
  readonly dismissFailure: () => void;
};

type PhotoUploadOptions = {
  readonly articleId: string;
  readonly onUploaded: () => void;
};

const idle: PhotoUploadPhase = { kind: 'idle' };

function appendPhoto(photo: ArticlePhoto) {
  return (current: readonly ArticlePhoto[] | undefined): readonly ArticlePhoto[] => [
    ...(current ?? []).filter(({ id }) => id !== photo.id),
    photo,
  ];
}

export function usePhotoUpload({ articleId, onUploaded }: PhotoUploadOptions): PhotoUploadModel {
  const client = usePhotosClient();
  const queryClient = useQueryClient();
  const [phase, setPhase] = useState<PhotoUploadPhase>(idle);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const controllerRef = useRef<AbortController | null>(null);
  const previewUrl = useObjectUrl(pendingFile);
  const isBusy = phase.kind === 'checking' || phase.kind === 'uploading';

  const cancel = useCallback(() => {
    controllerRef.current?.abort();
  }, []);

  useEffect(() => cancel, [cancel]);

  const upload = useCallback(
    async (file: File) => {
      if (client === null) {
        return;
      }
      setPhase({ kind: 'checking' });
      const controller = new AbortController();
      controllerRef.current = controller;
      try {
        const inspection = await inspectPhotoFile(file, client.limits);
        if (inspection.kind === 'rejected') {
          setPhase({ kind: 'failed', failure: inspection.reason });
          return;
        }
        setPendingFile(file);
        setPhase({ kind: 'uploading', progress: 0 });
        const photo = await client.uploadPhoto(articleId, {
          file,
          mediaType: inspection.mediaType,
          onProgress: (progress) => {
            setPhase({ kind: 'uploading', progress });
          },
          signal: controller.signal,
        });
        queryClient.setQueryData(photosQueryKeys.sellerArticle(articleId), appendPhoto(photo));
        setPhase(idle);
        onUploaded();
      } catch (error: unknown) {
        setPhase(isAbortError(error) ? idle : { kind: 'failed', failure: toPhotoFailure(error) });
      } finally {
        controllerRef.current = null;
        setPendingFile(null);
      }
    },
    [articleId, client, onUploaded, queryClient],
  );

  const selectFiles = useCallback(
    (files: readonly File[]) => {
      const [file] = files;
      if (file === undefined || isBusy) {
        return;
      }
      void upload(file);
    },
    [isBusy, upload],
  );

  const dismissFailure = useCallback(() => {
    setPhase(idle);
  }, []);

  return { phase, previewUrl, isBusy, selectFiles, cancel, dismissFailure };
}
