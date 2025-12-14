import React, { useState } from 'react';
import { Trash2, Star, GripVertical, Upload } from 'lucide-react';
import { useTranslation } from '@/i18n/provider';
import Image from 'next/image';
import photoService from '@/services/photo';
import type { ProductPhoto } from '@/types/photo';

interface PhotoManagerProps {
  productId: string;
  photos: ProductPhoto[];
  onPhotosChange: (photos: ProductPhoto[]) => void;
  maxPhotos?: number;
}

export default function PhotoManager({
  productId,
  photos,
  onPhotosChange,
  maxPhotos = 5,
}: PhotoManagerProps) {
  const { t } = useTranslation(['products', 'common']);
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [updating, setUpdating] = useState<string | null>(null);

  const sortedPhotos = [...photos].sort((a, b) => a.display_order - b.display_order);

  const handleDragStart = (index: number) => {
    setDraggedIndex(index);
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (draggedIndex === null || draggedIndex === index) return;

    const newPhotos = [...sortedPhotos];
    const draggedPhoto = newPhotos[draggedIndex];
    newPhotos.splice(draggedIndex, 1);
    newPhotos.splice(index, 0, draggedPhoto);

    // Update display orders
    const reordered = newPhotos.map((photo, idx) => ({
      ...photo,
      display_order: idx,
    }));

    onPhotosChange(reordered);
    setDraggedIndex(index);
  };

  const handleDragEnd = async () => {
    if (draggedIndex === null) return;

    try {
      // Send reorder request to backend
      await photoService.reorderPhotos(productId, {
        photos: sortedPhotos.map((photo, idx) => ({
          photo_id: photo.id,
          display_order: idx,
        })),
      });
    } catch (error) {
      console.error('Failed to reorder photos:', error);
    } finally {
      setDraggedIndex(null);
    }
  };

  const handleSetPrimary = async (photoId: string) => {
    setUpdating(photoId);
    try {
      await photoService.updatePhotoMetadata(productId, photoId, {
        isPrimary: true,
      });

      // Update local state
      const updated = photos.map(photo => ({
        ...photo,
        is_primary: photo.id === photoId,
      }));
      onPhotosChange(updated);
    } catch (error) {
      console.error('Failed to set primary photo:', error);
    } finally {
      setUpdating(null);
    }
  };

  const handleDelete = async (photoId: string) => {
    if (!confirm(t('products.form.confirmDeletePhoto'))) return;

    setDeleting(photoId);
    try {
      await photoService.deletePhoto(productId, photoId);
      onPhotosChange(photos.filter(p => p.id !== photoId));
    } catch (error) {
      console.error('Failed to delete photo:', error);
      alert(t('products.messages.deletePhotoError'));
    } finally {
      setDeleting(null);
    }
  };

  if (sortedPhotos.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        <Upload className="w-12 h-12 mx-auto mb-2 text-gray-400" />
        <p>{t('products.form.noPhotos')}</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {sortedPhotos.map((photo, index) => (
        <div
          key={photo.id}
          draggable
          onDragStart={() => handleDragStart(index)}
          onDragOver={(e) => handleDragOver(e, index)}
          onDragEnd={handleDragEnd}
          className={`flex items-center gap-4 p-3 bg-white border rounded-lg transition-all ${draggedIndex === index ? 'opacity-50 scale-95' : 'hover:shadow-md'
            } ${photo.is_primary ? 'border-blue-500 bg-blue-50' : 'border-gray-200'}`}
        >
          {/* Drag Handle */}
          <div className="cursor-move text-gray-400 hover:text-gray-600">
            <GripVertical className="w-5 h-5" />
          </div>

          {/* Photo Thumbnail */}
          <div className="relative w-16 h-16 rounded-md overflow-hidden bg-gray-100 flex-shrink-0">
            {photo.photo_url ? (
              <Image
                src={photo.photo_url}
                alt={photo.original_filename}
                fill
                className="object-cover"
                sizes="64px"
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center">
                <svg className="w-6 h-6 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </div>
            )}
          </div>

          {/* Photo Info */}
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-gray-900 truncate">
              {photo.original_filename}
            </p>
            <p className="text-xs text-gray-500">
              {photoService.formatFileSize(photo.file_size_bytes)}
              {photo.width_px && photo.height_px && ` • ${photo.width_px}×${photo.height_px}`}
            </p>
          </div>

          {/* Primary Badge */}
          {photo.is_primary && (
            <span className="px-2 py-1 text-xs font-medium text-blue-700 bg-blue-100 rounded-full">
              {t('products.form.primary')}
            </span>
          )}

          {/* Actions */}
          <div className="flex items-center gap-2">
            <button
              onClick={() => handleSetPrimary(photo.id)}
              disabled={photo.is_primary || updating !== null}
              className={`p-2 rounded-lg transition-colors ${photo.is_primary
                ? 'text-blue-500 cursor-default'
                : 'text-gray-400 hover:text-yellow-500 hover:bg-yellow-50'
                }`}
              title={t('products.form.setPrimary')}
            >
              <Star className={`w-5 h-5 ${photo.is_primary ? 'fill-current' : ''}`} />
            </button>

            <button
              onClick={() => handleDelete(photo.id)}
              disabled={deleting !== null}
              className="p-2 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors"
              title={t('products.form.deletePhoto')}
            >
              {deleting === photo.id ? (
                <div className="w-5 h-5 border-2 border-red-500 border-t-transparent rounded-full animate-spin" />
              ) : (
                <Trash2 className="w-5 h-5" />
              )}
            </button>
          </div>
        </div>
      ))}

      {/* Help Text */}
      <p className="text-xs text-gray-500 text-center">
        {t('products.form.dragToReorder')}
      </p>
    </div>
  );
}
