import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import BulkProductImportModal from './BulkProductImportModal';
import { product } from '@/services/product';

jest.mock('@/i18n/provider', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        'products.bulkImport.title': 'Bulk Product Import',
        'products.bulkImport.subtitle': 'Import products and stock from a CSV or XLSX file',
        'products.bulkImport.templateTitle': 'Template',
        'products.bulkImport.columnsTitle': 'Columns',
        'products.bulkImport.fileTitle': 'Import file',
        'products.bulkImport.downloadCSV': 'Download CSV',
        'products.bulkImport.downloadXLSX': 'Download XLSX',
        'products.bulkImport.selectFileButton': 'Choose CSV or XLSX file',
        'products.bulkImport.startButton': 'Start Import',
        'products.bulkImport.importingButton': 'Importing...',
        'products.bulkImport.doneButton': 'Done',
        'products.bulkImport.importId': 'Import ID',
        'products.bulkImport.summaryTitle': 'Summary',
        'products.bulkImport.errorsTitle': 'Row Errors',
        'products.bulkImport.warningsTitle': 'Warnings',
        'products.bulkImport.noErrors': 'No row errors',
        'products.bulkImport.noWarnings': 'No warnings',
        'products.bulkImport.progress.ready': 'Ready to import',
        'products.bulkImport.progress.uploading': 'Uploading file',
        'products.bulkImport.progress.processing': 'Processing import',
        'products.bulkImport.progress.completed': 'Import completed',
        'products.bulkImport.progress.failed': 'Import failed',
        'products.bulkImport.messages.fileRequired':
          'Choose a CSV or XLSX file before starting the import.',
        'products.bulkImport.messages.invalidFile': 'Only CSV and XLSX files are supported.',
        'products.bulkImport.messages.templateDownloadError':
          'Failed to download the template. Please try again.',
        'products.bulkImport.messages.importFailed':
          'Failed to import products. Please review the file and try again.',
        'products.bulkImport.messages.importCompleted': 'Products imported successfully.',
        'products.bulkImport.messages.importAccepted': 'Import completed.',
        'products.bulkImport.summary.totalRows': 'Total Rows',
        'products.bulkImport.summary.processedRows': 'Processed',
        'products.bulkImport.summary.importedRows': 'Imported',
        'products.bulkImport.summary.created': 'Created',
        'products.bulkImport.summary.updated': 'Updated',
        'products.bulkImport.summary.skipped': 'Skipped',
        'products.bulkImport.summary.failed': 'Failed',
        'products.bulkImport.summary.warnings': 'Warnings',
        'products.bulkImport.table.row': 'Row',
        'products.bulkImport.table.sku': 'SKU',
        'products.bulkImport.table.field': 'Field',
        'products.bulkImport.table.message': 'Message',
        'common.close': 'Close',
        'common.cancel': 'Cancel',
      };

      return translations[key] || key;
    },
  }),
}));

jest.mock('@/services/product', () => ({
  product: {
    downloadImportTemplate: jest.fn(),
    startProductImport: jest.fn(),
    pollProductImport: jest.fn(),
  },
}));

const mockProduct = product as jest.Mocked<typeof product>;

function renderModal(onImportComplete = jest.fn()) {
  return {
    onImportComplete,
    onClose: jest.fn(),
    ...render(
      <BulkProductImportModal isOpen onClose={jest.fn()} onImportComplete={onImportComplete} />
    ),
  };
}

beforeEach(() => {
  jest.clearAllMocks();
});

it('renders template downloads and accepts CSV or XLSX files', () => {
  renderModal();

  expect(screen.getByRole('dialog', { name: 'Bulk Product Import' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Download CSV' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Download XLSX' })).toBeInTheDocument();
  expect(screen.getByTestId('bulk-product-import-file-input')).toHaveAttribute(
    'accept',
    '.csv,.xlsx'
  );
});

it('rejects unsupported file types before upload', () => {
  renderModal();

  const input = screen.getByTestId('bulk-product-import-file-input');
  fireEvent.change(input, {
    target: {
      files: [new File(['sku,name'], 'products.txt', { type: 'text/plain' })],
    },
  });

  expect(screen.getByText('Only CSV and XLSX files are supported.')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Start Import' })).toBeDisabled();
  expect(mockProduct.startProductImport).not.toHaveBeenCalled();
});

it('uploads a file, polls status, and shows summary issues', async () => {
  const onImportComplete = jest.fn();
  renderModal(onImportComplete);

  mockProduct.startProductImport.mockImplementation(async (_file, options) => {
    options?.onUploadProgress?.(55);
    return { import_id: 'import-123', status: 'processing' };
  });

  mockProduct.pollProductImport.mockImplementation(async (_importId, options) => {
    const completed = {
      import_id: 'import-123',
      status: 'completed',
      summary: {
        total_rows: 2,
        created_count: 1,
        updated_count: 1,
      },
      row_errors: [
        {
          row: 2,
          sku: 'SKU-2',
          field: 'Selling Price',
          message: 'Selling Price must be greater than zero',
        },
      ],
      warnings: [
        {
          row: 1,
          sku: 'SKU-1',
          field: 'Photos',
          message: 'Photo URL could not be fetched',
        },
      ],
    };
    options?.onStatus?.(completed);
    return completed;
  });

  fireEvent.change(screen.getByTestId('bulk-product-import-file-input'), {
    target: {
      files: [new File(['SKU,Name'], 'products.csv', { type: 'text/csv' })],
    },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Start Import' }));

  await waitFor(() => {
    expect(onImportComplete).toHaveBeenCalledTimes(1);
  });

  expect(mockProduct.startProductImport).toHaveBeenCalledTimes(1);
  expect(mockProduct.pollProductImport).toHaveBeenCalledWith(
    'import-123',
    expect.objectContaining({ onStatus: expect.any(Function) })
  );
  expect(screen.getByText('Products imported successfully.')).toBeInTheDocument();
  expect(screen.getByText('Total Rows')).toBeInTheDocument();
  expect(screen.getByText('Selling Price must be greater than zero')).toBeInTheDocument();
  expect(screen.getByText('Photo URL could not be fetched')).toBeInTheDocument();
});
