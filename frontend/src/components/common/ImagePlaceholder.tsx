import React from 'react';
import Image from 'next/image';

interface ImagePlaceholderProps {
  src?: string | null;
  alt: string;
  width?: number;
  height?: number;
  fill?: boolean;
  className?: string;
  fallbackSrc?: string;
  onError?: () => void;
  sizes?: string;
}

export default function ImagePlaceholder({
  src,
  alt,
  width,
  height,
  fill = false,
  className = '',
  fallbackSrc = '/assets/placeholder-product.svg',
  onError,
  sizes,
}: ImagePlaceholderProps) {
  const [error, setError] = React.useState(false);
  const [loading, setLoading] = React.useState(true);

  const handleError = () => {
    setError(true);
    setLoading(false);
    onError?.();
  };

  const handleLoad = () => {
    setLoading(false);
  };

  // Use fallback if no src or error occurred
  const shouldUseFallback = !src || error;
  const imageSrc = shouldUseFallback ? fallbackSrc : src;

  return (
    <div className={`relative ${!fill ? 'inline-block' : 'w-full h-full'} ${className}`}>
      {loading && src && !error && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-100 rounded z-10">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      )}

      {fill ? (
        <Image
          src={imageSrc}
          alt={alt}
          fill
          className={className}
          onError={handleError}
          onLoad={handleLoad}
          sizes={sizes}
          unoptimized={shouldUseFallback}
        />
      ) : (
        <Image
          src={imageSrc}
          alt={alt}
          width={width || 200}
          height={height || 200}
          className={className}
          onError={handleError}
          onLoad={handleLoad}
          sizes={sizes}
          unoptimized={shouldUseFallback}
        />
      )}
    </div>
  );
}
