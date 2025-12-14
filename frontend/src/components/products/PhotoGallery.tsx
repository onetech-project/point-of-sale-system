import React, { useState } from 'react';
import { X, ChevronLeft, ChevronRight, ExternalLink } from 'lucide-react';
import Image from 'next/image';
import ImagePlaceholder from '@/components/common/ImagePlaceholder';
import type { ProductPhoto } from '@/types/photo';

interface PhotoGalleryProps {
  photos: ProductPhoto[];
  className?: string;
}

export default function PhotoGallery({ photos, className = '' }: PhotoGalleryProps) {
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);

  if (!photos || photos.length === 0) {
    return null;
  }

  const openLightbox = (index: number) => {
    setSelectedIndex(index);
  };

  const closeLightbox = () => {
    setSelectedIndex(null);
  };

  const goToPrevious = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (selectedIndex !== null && selectedIndex > 0) {
      setSelectedIndex(selectedIndex - 1);
    }
  };

  const goToNext = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (selectedIndex !== null && selectedIndex < photos.length - 1) {
      setSelectedIndex(selectedIndex + 1);
    }
  };

  // Sort photos by display order
  const sortedPhotos = [...photos].sort((a, b) => a.display_order - b.display_order);

  if (sortedPhotos.length === 0) {
    return null;
  }

  const primaryPhoto = sortedPhotos.find(p => p.is_primary) || sortedPhotos[0];

  return (
    <>
      <div className={className}>
        {/* Main Photo Display */}
        <div
          className="relative w-full aspect-square bg-gray-100 rounded-lg overflow-hidden mb-3 cursor-pointer group"
          style={{ minHeight: '300px' }}
          onClick={() => openLightbox(sortedPhotos.indexOf(primaryPhoto))}
        >
          {primaryPhoto.photo_url ? (
            <ImagePlaceholder
              src={primaryPhoto.photo_url}
              alt={primaryPhoto.original_filename}
              fill
              className="object-contain hover:opacity-90 transition-opacity"
              sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw"
            />
          ) : (
            <div className="absolute inset-0 flex items-center justify-center bg-gray-50">
              <div className="text-center text-gray-400">
                <svg className="w-16 h-16 mx-auto mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                <p className="text-sm">Photo not available</p>
              </div>
            </div>
          )}
        </div>

        {/* Thumbnail Strip */}
        {sortedPhotos.length > 1 && (
          <div className="flex gap-2 overflow-x-auto pb-2">
            {sortedPhotos.map((photo, index) => (
              <button
                key={photo.id}
                onClick={() => openLightbox(index)}
                className={`relative flex-shrink-0 w-20 h-20 rounded-md overflow-hidden border-2 transition-all ${photo.id === primaryPhoto.id
                  ? 'border-blue-500 ring-2 ring-blue-200'
                  : 'border-gray-200 hover:border-gray-300'
                  }`}
              >
                {photo.photo_url ? (
                  <ImagePlaceholder
                    src={photo.photo_url}
                    alt={photo.original_filename}
                    fill
                    className="object-cover"
                    sizes="80px"
                  />
                ) : (
                  <div className="absolute inset-0 flex items-center justify-center bg-gray-100">
                    <svg className="w-8 h-8 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                  </div>
                )}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Lightbox Modal */}
      {selectedIndex !== null && (
        <div
          className="fixed inset-0 z-50 bg-black bg-opacity-90 flex items-center justify-center"
          onClick={closeLightbox}
        >
          <button
            onClick={closeLightbox}
            className="absolute top-4 right-4 text-white hover:text-gray-300 transition-colors"
          >
            <X className="w-8 h-8" />
          </button>

          {/* Navigation Arrows */}
          {selectedIndex > 0 && (
            <button
              onClick={goToPrevious}
              className="absolute left-4 text-white hover:text-gray-300 transition-colors"
            >
              <ChevronLeft className="w-12 h-12" />
            </button>
          )}

          {selectedIndex < sortedPhotos.length - 1 && (
            <button
              onClick={goToNext}
              className="absolute right-4 text-white hover:text-gray-300 transition-colors"
            >
              <ChevronRight className="w-12 h-12" />
            </button>
          )}

          {/* Image Display */}
          <div className="relative max-w-7xl max-h-[90vh] mx-auto" onClick={(e) => e.stopPropagation()}>
            <div className="relative">
              <ImagePlaceholder
                src={sortedPhotos[selectedIndex].photo_url || ''}
                alt={sortedPhotos[selectedIndex].original_filename}
                width={1200}
                height={1200}
                className="max-h-[90vh] w-auto object-contain"
              />
              {sortedPhotos[selectedIndex].photo_url && (
                <a
                  href={sortedPhotos[selectedIndex].photo_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="absolute bottom-4 right-4 bg-white bg-opacity-90 hover:bg-opacity-100 text-gray-800 px-3 py-2 rounded-lg flex items-center gap-2 text-sm transition-colors"
                  onClick={(e) => e.stopPropagation()}
                >
                  <ExternalLink className="w-4 h-4" />
                  Open Original
                </a>
              )}
            </div>
          </div>

          {/* Image Counter */}
          <div className="absolute bottom-4 left-1/2 transform -translate-x-1/2 text-white text-sm bg-black bg-opacity-50 px-4 py-2 rounded-full">
            {selectedIndex + 1} / {sortedPhotos.length}
          </div>
        </div>
      )}
    </>
  );
}
