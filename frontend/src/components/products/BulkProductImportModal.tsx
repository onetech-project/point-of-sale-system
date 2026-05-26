'use client';

import React, { useEffect, useRef, useState } from 'react';
import {
  AlertTriangle,
  CheckCircle2,
  Download,
  FileSpreadsheet,
  Loader2,
  Upload,
  X,
} from 'lucide-react';
import { useTranslation } from '@/i18n/provider';
import { product } from '@/services/product';
import {
  ProductImportRowIssue,
  ProductImportStatus,
  ProductImportStatusResponse,
  ProductImportSummary,
  ProductImportTemplateFormat,
} from '@/types/product';

type ImportPhase = 'idle' | 'uploading' | 'polling' | 'completed' | 'failed';

interface BulkProductImportModalProps {
  isOpen: boolean;
  onClose: () => void;
  onImportComplete: () => void;
}

const SUCCESS_STATUSES = new Set<ProductImportStatus>([
  'completed',
  'success',
  'succeeded',
  'completed_with_errors',
  'partial_success',
]);

const FAILURE_STATUSES = new Set<ProductImportStatus>(['failed', 'error', 'cancelled', 'canceled']);

const TEMPLATE_HEADERS = [
  'SKU',
  'Name',
  'Category',
  'Selling Price',
  'Cost Price',
  'Tax Rate',
  'Stock Quantity',
  'Description',
  'Photos',
];

const getStatusKey = (status?: ProductImportStatus) =>
  String(status || '').toLowerCase() as ProductImportStatus;

const isSuccessStatus = (status?: ProductImportStatus) =>
  SUCCESS_STATUSES.has(getStatusKey(status));
const isFailureStatus = (status?: ProductImportStatus) =>
  FAILURE_STATUSES.has(getStatusKey(status));

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') {
    return false;
  }

  const maybeError = error as { name?: string; code?: string };
  return (
    maybeError.name === 'AbortError' ||
    maybeError.name === 'CanceledError' ||
    maybeError.code === 'ERR_CANCELED'
  );
};

const displayValue = (value: unknown) => {
  if (value === null || value === undefined || value === '') {
    return '-';
  }

  return String(value);
};

const getIssueField = (issue: ProductImportRowIssue, keys: string[]) => {
  for (const key of keys) {
    const value = issue[key];
    if (value !== null && value !== undefined && value !== '') {
      return value;
    }
  }

  return undefined;
};

const getSummaryValue = (summary: ProductImportSummary | null | undefined, keys: string[]) => {
  if (!summary) {
    return null;
  }

  for (const key of keys) {
    const value = summary[key];
    if (value !== null && value !== undefined && value !== '') {
      return value;
    }
  }

  return null;
};

const humanizeSummaryKey = (key: string) =>
  key.replace(/_/g, ' ').replace(/\b\w/g, letter => letter.toUpperCase());

export default function BulkProductImportModal({
  isOpen,
  onClose,
  onImportComplete,
}: BulkProductImportModalProps) {
  const { t } = useTranslation(['products', 'common']);
  const abortRef = useRef<AbortController | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [phase, setPhase] = useState<ImportPhase>('idle');
  const [downloadFormat, setDownloadFormat] = useState<ProductImportTemplateFormat | null>(null);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [importId, setImportId] = useState<string | null>(null);
  const [importStatus, setImportStatus] = useState<ProductImportStatus | null>(null);
  const [result, setResult] = useState<ProductImportStatusResponse | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const busy = phase === 'uploading' || phase === 'polling';
  const rowErrors = result?.row_errors || result?.errors || [];
  const warnings = result?.warnings || [];

  useEffect(() => {
    if (isOpen) {
      setSelectedFile(null);
      setPhase('idle');
      setDownloadFormat(null);
      setUploadProgress(0);
      setImportId(null);
      setImportStatus(null);
      setResult(null);
      setErrorMessage(null);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    } else {
      abortRef.current?.abort();
      abortRef.current = null;
    }

    return () => {
      abortRef.current?.abort();
      abortRef.current = null;
    };
  }, [isOpen]);

  if (!isOpen) {
    return null;
  }

  const handleTemplateDownload = async (format: ProductImportTemplateFormat) => {
    try {
      setDownloadFormat(format);
      setErrorMessage(null);
      const template = await product.downloadImportTemplate(format);
      const url = window.URL.createObjectURL(template.blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = template.filename;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
    } catch (err: any) {
      console.error('Failed to download product import template:', err);
      setErrorMessage(
        err.response?.data?.message || t('products.bulkImport.messages.templateDownloadError')
      );
    } finally {
      setDownloadFormat(null);
    }
  };

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] || null;
    setErrorMessage(null);
    setResult(null);
    setImportId(null);
    setImportStatus(null);

    if (!file) {
      setSelectedFile(null);
      return;
    }

    const lowerName = file.name.toLowerCase();
    if (!lowerName.endsWith('.csv') && !lowerName.endsWith('.xlsx')) {
      setSelectedFile(null);
      event.target.value = '';
      setErrorMessage(t('products.bulkImport.messages.invalidFile'));
      return;
    }

    setSelectedFile(file);
    setPhase('idle');
  };

  const handleStartImport = async () => {
    if (!selectedFile) {
      setErrorMessage(t('products.bulkImport.messages.fileRequired'));
      return;
    }

    const controller = new AbortController();
    abortRef.current?.abort();
    abortRef.current = controller;

    try {
      setErrorMessage(null);
      setResult(null);
      setUploadProgress(0);
      setPhase('uploading');

      const startedImport = await product.startProductImport(selectedFile, {
        signal: controller.signal,
        onUploadProgress: progress => setUploadProgress(progress),
      });

      setImportId(startedImport.import_id);
      setImportStatus(startedImport.status);
      setUploadProgress(100);
      setPhase('polling');

      const completedImport = await product.pollProductImport(startedImport.import_id, {
        signal: controller.signal,
        onStatus: status => {
          setResult(status);
          setImportStatus(status.status);
        },
      });

      setResult(completedImport);
      setImportStatus(completedImport.status);

      if (isFailureStatus(completedImport.status)) {
        setPhase('failed');
        setErrorMessage(completedImport.message || t('products.bulkImport.messages.importFailed'));
        return;
      }

      setPhase('completed');
      onImportComplete();
    } catch (err: any) {
      if (isAbortError(err)) {
        return;
      }

      console.error('Failed to import products:', err);
      setPhase('failed');
      setErrorMessage(
        err.response?.data?.message || err.message || t('products.bulkImport.messages.importFailed')
      );
    } finally {
      abortRef.current = null;
    }
  };

  const handleClose = () => {
    abortRef.current?.abort();
    abortRef.current = null;
    onClose();
  };

  const summaryItems = buildSummaryItems(result?.summary, warnings.length, t);
  const progressLabel =
    phase === 'uploading'
      ? t('products.bulkImport.progress.uploading')
      : phase === 'polling'
        ? t('products.bulkImport.progress.processing')
        : phase === 'completed'
          ? t('products.bulkImport.progress.completed')
          : phase === 'failed'
            ? t('products.bulkImport.progress.failed')
            : t('products.bulkImport.progress.ready');

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-gray-900/60 px-4 py-6"
      role="dialog"
      aria-modal="true"
      aria-labelledby="bulk-product-import-title"
    >
      <div className="flex max-h-full w-full max-w-4xl flex-col overflow-hidden rounded-lg bg-white shadow-xl">
        <div className="flex items-start justify-between gap-4 border-b border-gray-200 px-5 py-4">
          <div className="min-w-0">
            <h2
              id="bulk-product-import-title"
              className="text-lg font-semibold leading-6 text-gray-900"
            >
              {t('products.bulkImport.title')}
            </h2>
            <p className="mt-1 text-sm text-gray-500">{t('products.bulkImport.subtitle')}</p>
          </div>
          <button
            type="button"
            onClick={handleClose}
            className="rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 focus:outline-none focus:ring-2 focus:ring-primary-500"
            aria-label={t('common.close', { ns: 'common' })}
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        <div className="overflow-y-auto px-5 py-4">
          <div className="grid gap-5 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
            <section className="space-y-4">
              <div>
                <h3 className="text-sm font-semibold text-gray-900">
                  {t('products.bulkImport.templateTitle')}
                </h3>
                <div className="mt-3 flex flex-wrap gap-2">
                  {(['csv', 'xlsx'] as ProductImportTemplateFormat[]).map(format => (
                    <button
                      key={format}
                      type="button"
                      onClick={() => handleTemplateDownload(format)}
                      disabled={downloadFormat !== null}
                      className="inline-flex items-center gap-2 rounded-md border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                      {downloadFormat === format ? (
                        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                      ) : (
                        <Download className="h-4 w-4" aria-hidden="true" />
                      )}
                      {t(`products.bulkImport.download${format.toUpperCase()}`)}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <h3 className="text-sm font-semibold text-gray-900">
                  {t('products.bulkImport.columnsTitle')}
                </h3>
                <div className="mt-2 flex flex-wrap gap-2">
                  {TEMPLATE_HEADERS.map(header => (
                    <span
                      key={header}
                      className="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700"
                    >
                      {header}
                    </span>
                  ))}
                </div>
              </div>

              <div>
                <h3 className="text-sm font-semibold text-gray-900">
                  {t('products.bulkImport.fileTitle')}
                </h3>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".csv,.xlsx"
                  className="sr-only"
                  onChange={handleFileChange}
                  disabled={busy}
                  data-testid="bulk-product-import-file-input"
                />
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={busy}
                  className="mt-3 flex w-full items-center justify-center gap-3 rounded-lg border-2 border-dashed border-gray-300 px-4 py-8 text-sm font-medium text-gray-700 hover:border-primary-400 hover:bg-primary-50 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <FileSpreadsheet className="h-6 w-6 text-primary-600" aria-hidden="true" />
                  <span>
                    {selectedFile ? selectedFile.name : t('products.bulkImport.selectFileButton')}
                  </span>
                </button>
              </div>
            </section>

            <section className="space-y-4">
              <div aria-live="polite">
                <div className="flex items-center justify-between gap-3 text-sm">
                  <span className="font-medium text-gray-900">{progressLabel}</span>
                  {importStatus && (
                    <span className="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium uppercase tracking-wide text-gray-600">
                      {importStatus}
                    </span>
                  )}
                </div>
                <div className="mt-3 h-2 overflow-hidden rounded-full bg-gray-200">
                  <div
                    className={`h-full rounded-full transition-all ${
                      phase === 'failed'
                        ? 'bg-red-600'
                        : phase === 'completed'
                          ? 'bg-green-600'
                          : 'bg-primary-600'
                    } ${phase === 'polling' ? 'animate-pulse' : ''}`}
                    style={{
                      width:
                        phase === 'uploading'
                          ? `${uploadProgress}%`
                          : phase === 'idle'
                            ? '0%'
                            : '100%',
                    }}
                    data-testid="bulk-product-import-progress"
                  />
                </div>
                {importId && (
                  <p className="mt-2 text-xs text-gray-500">
                    {t('products.bulkImport.importId')}: {importId}
                  </p>
                )}
              </div>

              {errorMessage && (
                <div className="flex gap-3 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
                  <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0" aria-hidden="true" />
                  <p>{errorMessage}</p>
                </div>
              )}

              {phase === 'completed' && (
                <div className="flex gap-3 rounded-md border border-green-200 bg-green-50 p-3 text-sm text-green-800">
                  <CheckCircle2 className="mt-0.5 h-5 w-5 flex-shrink-0" aria-hidden="true" />
                  <p>
                    {isSuccessStatus(result?.status)
                      ? t('products.bulkImport.messages.importCompleted')
                      : t('products.bulkImport.messages.importAccepted')}
                  </p>
                </div>
              )}

              {summaryItems.length > 0 && (
                <div>
                  <h3 className="text-sm font-semibold text-gray-900">
                    {t('products.bulkImport.summaryTitle')}
                  </h3>
                  <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-3 sm:grid-cols-3">
                    {summaryItems.map(item => (
                      <div key={item.label} className="min-w-0 border-l border-gray-200 pl-3">
                        <dt className="truncate text-xs font-medium text-gray-500">{item.label}</dt>
                        <dd className="mt-1 text-lg font-semibold text-gray-900">
                          {displayValue(item.value)}
                        </dd>
                      </div>
                    ))}
                  </dl>
                </div>
              )}

              <IssueTable
                title={t('products.bulkImport.errorsTitle')}
                emptyLabel={t('products.bulkImport.noErrors')}
                issues={rowErrors}
                tone="error"
                t={t}
              />

              <IssueTable
                title={t('products.bulkImport.warningsTitle')}
                emptyLabel={t('products.bulkImport.noWarnings')}
                issues={warnings}
                tone="warning"
                t={t}
              />
            </section>
          </div>
        </div>

        <div className="flex flex-col-reverse gap-2 border-t border-gray-200 px-5 py-4 sm:flex-row sm:justify-end">
          <button
            type="button"
            onClick={handleClose}
            className="inline-flex items-center justify-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            {phase === 'completed'
              ? t('products.bulkImport.doneButton')
              : t('common.cancel', { ns: 'common' })}
          </button>
          <button
            type="button"
            onClick={handleStartImport}
            disabled={!selectedFile || busy}
            className="inline-flex items-center justify-center gap-2 rounded-md bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {busy ? (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            ) : (
              <Upload className="h-4 w-4" aria-hidden="true" />
            )}
            {busy ? t('products.bulkImport.importingButton') : t('products.bulkImport.startButton')}
          </button>
        </div>
      </div>
    </div>
  );
}

function buildSummaryItems(
  summary: ProductImportSummary | null | undefined,
  warningCount: number,
  t: (key: string) => string
) {
  const knownItems = [
    {
      label: t('products.bulkImport.summary.totalRows'),
      value: getSummaryValue(summary, ['total_rows', 'total', 'total_count']),
    },
    {
      label: t('products.bulkImport.summary.processedRows'),
      value: getSummaryValue(summary, ['processed_rows', 'processed', 'processed_count']),
    },
    {
      label: t('products.bulkImport.summary.importedRows'),
      value: getSummaryValue(summary, ['imported_rows', 'imported_count', 'imported']),
    },
    {
      label: t('products.bulkImport.summary.created'),
      value: getSummaryValue(summary, ['created_count', 'created']),
    },
    {
      label: t('products.bulkImport.summary.updated'),
      value: getSummaryValue(summary, ['updated_count', 'updated']),
    },
    {
      label: t('products.bulkImport.summary.skipped'),
      value: getSummaryValue(summary, ['skipped_count', 'skipped']),
    },
    {
      label: t('products.bulkImport.summary.failed'),
      value: getSummaryValue(summary, ['failed_rows', 'failed_count', 'error_count']),
    },
    {
      label: t('products.bulkImport.summary.warnings'),
      value: warningCount || getSummaryValue(summary, ['warning_count', 'warnings_count']),
    },
  ].filter(item => item.value !== null && item.value !== undefined && item.value !== '');

  if (knownItems.length > 0 || !summary) {
    return knownItems;
  }

  return Object.entries(summary)
    .filter(([, value]) => value !== null && value !== undefined && typeof value !== 'object')
    .slice(0, 8)
    .map(([key, value]) => ({
      label: humanizeSummaryKey(key),
      value,
    }));
}

function IssueTable({
  title,
  emptyLabel,
  issues,
  tone,
  t,
}: {
  title: string;
  emptyLabel: string;
  issues: ProductImportRowIssue[];
  tone: 'error' | 'warning';
  t: (key: string) => string;
}) {
  const titleColor = tone === 'error' ? 'text-red-700' : 'text-yellow-700';

  return (
    <div>
      <div className="flex items-center justify-between gap-3">
        <h3 className={`text-sm font-semibold ${titleColor}`}>{title}</h3>
        <span className="text-xs font-medium text-gray-500">{issues.length}</span>
      </div>
      {issues.length === 0 ? (
        <p className="mt-2 text-sm text-gray-500">{emptyLabel}</p>
      ) : (
        <div className="mt-2 max-h-56 overflow-auto rounded-md border border-gray-200">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="sticky top-0 bg-gray-50">
              <tr>
                <th className="px-3 py-2 text-left text-xs font-semibold uppercase text-gray-500">
                  {t('products.bulkImport.table.row')}
                </th>
                <th className="px-3 py-2 text-left text-xs font-semibold uppercase text-gray-500">
                  {t('products.bulkImport.table.sku')}
                </th>
                <th className="px-3 py-2 text-left text-xs font-semibold uppercase text-gray-500">
                  {t('products.bulkImport.table.field')}
                </th>
                <th className="px-3 py-2 text-left text-xs font-semibold uppercase text-gray-500">
                  {t('products.bulkImport.table.message')}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 bg-white">
              {issues.map((issue, index) => (
                <tr
                  key={`${displayValue(getIssueField(issue, ['row', 'row_number', 'line']))}-${index}`}
                >
                  <td className="whitespace-nowrap px-3 py-2 text-gray-700">
                    {displayValue(getIssueField(issue, ['row', 'row_number', 'line']))}
                  </td>
                  <td className="whitespace-nowrap px-3 py-2 text-gray-700">
                    {displayValue(getIssueField(issue, ['sku', 'SKU']))}
                  </td>
                  <td className="whitespace-nowrap px-3 py-2 text-gray-700">
                    {displayValue(getIssueField(issue, ['field', 'column']))}
                  </td>
                  <td className="min-w-64 px-3 py-2 text-gray-700">
                    {displayValue(getIssueField(issue, ['message', 'error', 'warning', 'reason']))}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
