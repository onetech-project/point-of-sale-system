'use client';

import React, { useEffect, useState } from 'react';
import type { CompletedOrderCorrection } from '../../types/offlineOrder';

const MIN_REASON_LENGTH = 5;
const MAX_REASON_LENGTH = 500;

interface AccountingImpactConfirmModalProps {
  isOpen: boolean;
  mode: 'edit' | 'delete';
  orderReference: string;
  /** Short lines describing what will change, shown above the reason field */
  impactSummary?: string[];
  isSubmitting: boolean;
  error?: string | null;
  onConfirm: (correction: CompletedOrderCorrection) => void;
  onCancel: () => void;
}

/**
 * Confirmation for owner corrections of COMPLETE orders.
 * A completed order is already part of the sales records, so the owner has to give a reason and
 * explicitly acknowledge that the change affects those records before it is submitted.
 */
export const AccountingImpactConfirmModal: React.FC<AccountingImpactConfirmModalProps> = ({
  isOpen,
  mode,
  orderReference,
  impactSummary = [],
  isSubmitting,
  error,
  onConfirm,
  onCancel,
}) => {
  const [reason, setReason] = useState('');
  const [acknowledged, setAcknowledged] = useState(false);

  useEffect(() => {
    if (!isOpen) {
      setReason('');
      setAcknowledged(false);
    }
  }, [isOpen]);

  if (!isOpen) {
    return null;
  }

  const trimmedReason = reason.trim();
  const canSubmit =
    acknowledged &&
    trimmedReason.length >= MIN_REASON_LENGTH &&
    trimmedReason.length <= MAX_REASON_LENGTH &&
    !isSubmitting;

  const isDelete = mode === 'delete';
  const title = isDelete ? 'Hapus Order yang Sudah Selesai' : 'Simpan Perubahan Order yang Sudah Selesai';
  const confirmLabel = isDelete ? 'Ya, Hapus Order' : 'Ya, Simpan Perubahan';

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    onConfirm({ reason: trimmedReason, acknowledge_accounting_impact: true });
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" role="dialog" aria-modal="true" aria-labelledby="accounting-impact-title">
      <div className="fixed inset-0 bg-black bg-opacity-50" onClick={isSubmitting ? undefined : onCancel} />

      <div className="flex min-h-full items-center justify-center p-4">
        <div className="relative bg-white rounded-lg shadow-xl max-w-lg w-full">
          <div className="px-6 py-4 border-b border-gray-200">
            <h3 id="accounting-impact-title" className="text-lg font-semibold text-gray-900">
              {title}
            </h3>
            <p className="text-sm text-gray-600">
              Order: <span className="font-mono">{orderReference}</span>
            </p>
          </div>

          <form onSubmit={handleSubmit}>
            <div className="px-6 py-4 space-y-4">
              <div className="bg-red-50 border border-red-200 rounded-md p-3">
                <p className="text-sm font-semibold text-red-800">
                  Perubahan ini akan memengaruhi pencatatan
                </p>
                <ul className="mt-2 text-sm text-red-700 list-disc pl-5 space-y-1">
                  {isDelete ? (
                    <>
                      <li>Order dikeluarkan dari laporan penjualan dan pendapatan.</li>
                      <li>Stok produk pada order ini dikembalikan.</li>
                      <li>Pembayaran yang sudah tercatat tidak dibatalkan otomatis; lakukan pengembalian dana secara manual bila perlu.</li>
                    </>
                  ) : (
                    <>
                      <li>Total order dan laporan pendapatan dihitung ulang.</li>
                      <li>Stok produk disesuaikan dengan item yang baru.</li>
                      <li>Pembayaran yang sudah tercatat tidak berubah; selisih dengan total baru dicatat di audit trail.</li>
                    </>
                  )}
                  <li>Tindakan ini dicatat di audit trail beserta nama Anda dan alasannya.</li>
                </ul>
                {impactSummary.length > 0 && (
                  <ul className="mt-3 text-sm text-red-900 font-medium space-y-1">
                    {impactSummary.map(line => (
                      <li key={line}>{line}</li>
                    ))}
                  </ul>
                )}
              </div>

              <div>
                <label htmlFor="correction-reason" className="block text-sm font-medium text-gray-700 mb-2">
                  Alasan {isDelete ? 'penghapusan' : 'perubahan'} *
                </label>
                <textarea
                  id="correction-reason"
                  value={reason}
                  onChange={e => setReason(e.target.value)}
                  placeholder="Contoh: jumlah item salah input saat transaksi"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500 min-h-[90px] resize-y"
                  disabled={isSubmitting}
                  maxLength={MAX_REASON_LENGTH}
                  required
                />
                <div className="mt-1 flex justify-between text-xs text-gray-500">
                  <span>Minimal {MIN_REASON_LENGTH} karakter</span>
                  <span>
                    {reason.length}/{MAX_REASON_LENGTH}
                  </span>
                </div>
              </div>

              <label className="flex items-start gap-2 text-sm text-gray-800">
                <input
                  type="checkbox"
                  checked={acknowledged}
                  onChange={e => setAcknowledged(e.target.checked)}
                  disabled={isSubmitting}
                  className="mt-0.5 h-4 w-4 rounded border-gray-300 text-red-600 focus:ring-red-500"
                />
                <span>
                  Saya memahami bahwa {isDelete ? 'penghapusan' : 'perubahan'} order yang sudah selesai akan
                  memengaruhi pencatatan penjualan, pendapatan, dan stok.
                </span>
              </label>

              {error && (
                <p className="text-sm text-red-600" role="alert">
                  {error}
                </p>
              )}
            </div>

            <div className="px-6 py-4 bg-gray-50 border-t border-gray-200 flex justify-end gap-3 rounded-b-lg">
              <button
                type="button"
                onClick={onCancel}
                disabled={isSubmitting}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
              >
                Batal
              </button>
              <button
                type="submit"
                disabled={!canSubmit}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isSubmitting ? 'Memproses...' : confirmLabel}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default AccountingImpactConfirmModal;
