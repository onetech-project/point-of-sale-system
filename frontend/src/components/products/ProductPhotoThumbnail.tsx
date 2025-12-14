import React from 'react';
import Image from 'next/image';
import type { ProductPhoto } from '@/types/photo';

interface ProductPhotoThumbnailProps {
  photo?: ProductPhoto | null;
  productName: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export default function ProductPhotoThumbnail({
  photo,
  productName,
  size = 'md',
  className = '',
}: ProductPhotoThumbnailProps) {
  const sizeClasses = {
    sm: 'w-12 h-12',
    md: 'w-16 h-16',
    lg: 'w-24 h-24',
  };

  const iconSizes = {
    sm: 'w-4 h-4',
    md: 'w-6 h-6',
    lg: 'w-10 h-10',
  };

  return (
    <div
      className={`${sizeClasses[size]} ${className} relative rounded-md overflow-hidden bg-gray-100 flex-shrink-0`}
    >
      {photo?.photo_url ? (
        <Image
          src={photo.photo_url}
          alt={productName}
          fill
          className="object-cover"
          sizes={size === 'sm' ? '48px' : size === 'md' ? '64px' : '96px'}
        />
      ) : (
        <div className="w-full h-full flex items-center justify-center">
          <svg
            className={`${iconSizes[size]} text-gray-400`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
            />
          </svg>
        </div>
      )}
    </div>
  );
}
