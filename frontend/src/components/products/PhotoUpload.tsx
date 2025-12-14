import React, { useState, useRef, ChangeEvent } from 'react';
import { Upload, X, Image as ImageIcon, AlertCircle } from 'lucide-react';
import photoService, { ProductPhoto } from '@/services/photoService';

interface PhotoUploadProps {
  productId?: string;
  maxPhotos?: number;
  existingPhotos?: ProductPhoto[];
  onUploadSuccess?: (photo: ProductPhoto) => void;
  onUploadError?: (error: string) => void;
  className?: string;
}

interface UploadProgress {
  fileName: string;
  progress: number;
  error?: string;
}

export default function PhotoUpload({
  productId,
  maxPhotos = 5,
  existingPhotos = [],
  onUploadSuccess,
  onUploadError,
  className = '',
}: PhotoUploadProps) {
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<UploadProgress | null>(null);
  const [dragActive, setDragActive] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const canUploadMore = existingPhotos.length < maxPhotos;

  const handleFileSelect = async (files: FileList | null) => {
    if (!files || files.length === 0) return;

    // Only process first file
    const file = files[0];
    await uploadFile(file);
  };

  const uploadFile = async (file: File) => {
    setError(null);
    setUploadProgress(null);

    // Validate file
    const validation = photoService.validateFile(file);
    if (!validation.valid) {
      const errorMsg = validation.error || 'Invalid file';
      setError(errorMsg);
      onUploadError?.(errorMsg);
      return;
    }

    // Check if we can upload more
    if (!canUploadMore) {
      const errorMsg = `Maximum ${maxPhotos} photos allowed per product`;
      setError(errorMsg);
      onUploadError?.(errorMsg);
      return;
    }

    // Check if productId is provided
    if (!productId) {
      const errorMsg = 'Product must be saved before uploading photos';
      setError(errorMsg);
      onUploadError?.(errorMsg);
      return;
    }

    setUploading(true);
    setUploadProgress({
      fileName: file.name,
      progress: 0,
    });

    try {
      const photo = await photoService.uploadPhoto({
        productId,
        file,
        displayOrder: existingPhotos.length,
        isPrimary: existingPhotos.length === 0, // First photo is primary by default
        onProgress: (progress) => {
          setUploadProgress({
            fileName: file.name,
            progress,
          });
        },
      });

      // Success
      setUploadProgress({
        fileName: file.name,
        progress: 100,
      });

      // Clear progress after a short delay
      setTimeout(() => {
        setUploadProgress(null);
      }, 1000);

      onUploadSuccess?.(photo);
    } catch (err: any) {
      const errorMsg = err.response?.data?.error?.message || 'Failed to upload photo';
      setError(errorMsg);
      setUploadProgress({
        fileName: file.name,
        progress: 0,
        error: errorMsg,
      });
      onUploadError?.(errorMsg);

      // Clear error after 5 seconds
      setTimeout(() => {
        setUploadProgress(null);
        setError(null);
      }, 5000);
    } finally {
      setUploading(false);
      // Reset file input
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    handleFileSelect(e.target.files);
  };

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFileSelect(e.dataTransfer.files);
    }
  };

  const handleClick = () => {
    fileInputRef.current?.click();
  };

  return (
    <div className={className}>
      {/* Upload Area */}
      <div
        className={`
          relative border-2 border-dashed rounded-lg p-6 text-center
          transition-colors duration-200
          ${dragActive ? 'border-blue-500 bg-blue-50' : 'border-gray-300 bg-gray-50'}
          ${!canUploadMore || !productId || uploading ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer hover:border-blue-400 hover:bg-blue-50'}
        `}
        onDragEnter={handleDrag}
        onDragLeave={handleDrag}
        onDragOver={handleDrag}
        onDrop={handleDrop}
        onClick={canUploadMore && productId && !uploading ? handleClick : undefined}
      >
        <input
          ref={fileInputRef}
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          onChange={handleInputChange}
          disabled={!canUploadMore || !productId || uploading}
          className="hidden"
        />

        <div className="flex flex-col items-center space-y-3">
          {uploading ? (
            <>
              <Upload className="w-12 h-12 text-blue-500 animate-pulse" />
              <div className="text-sm font-medium text-gray-700">Uploading...</div>
            </>
          ) : (
            <>
              <ImageIcon className="w-12 h-12 text-gray-400" />
              <div className="text-sm text-gray-600">
                {!productId ? (
                  <span className="text-orange-600 font-medium">Save product first to upload photos</span>
                ) : !canUploadMore ? (
                  <span className="text-orange-600 font-medium">Maximum {maxPhotos} photos reached</span>
                ) : (
                  <>
                    <span className="font-medium text-blue-600">Click to upload</span> or drag and drop
                  </>
                )}
              </div>
              {productId && canUploadMore && (
                <div className="text-xs text-gray-500">
                  JPEG, PNG, GIF, or WebP (max 10MB)
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Upload Progress */}
      {uploadProgress && (
        <div className="mt-4 p-4 bg-white border border-gray-200 rounded-lg">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-700 truncate flex-1">
              {uploadProgress.fileName}
            </span>
            {uploadProgress.error && (
              <AlertCircle className="w-5 h-5 text-red-500 ml-2 flex-shrink-0" />
            )}
          </div>
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div
              className={`h-2 rounded-full transition-all duration-300 ${uploadProgress.error ? 'bg-red-500' : 'bg-blue-500'
                }`}
              style={{ width: `${uploadProgress.progress}%` }}
            />
          </div>
          {uploadProgress.error && (
            <p className="mt-2 text-sm text-red-600">{uploadProgress.error}</p>
          )}
          {uploadProgress.progress === 100 && !uploadProgress.error && (
            <p className="mt-2 text-sm text-green-600">Upload complete!</p>
          )}
        </div>
      )}

      {/* General Error Message */}
      {error && !uploadProgress && (
        <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start">
          <AlertCircle className="w-5 h-5 text-red-500 mr-3 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-sm font-medium text-red-800">Upload Failed</p>
            <p className="text-sm text-red-700 mt-1">{error}</p>
          </div>
          <button
            onClick={() => setError(null)}
            className="text-red-500 hover:text-red-700 ml-2"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      )}

      {/* Photo Count Info */}
      {productId && (
        <div className="mt-3 text-xs text-gray-500 text-center">
          {existingPhotos.length} of {maxPhotos} photos uploaded
        </div>
      )}
    </div>
  );
}
