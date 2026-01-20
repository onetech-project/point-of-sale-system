import React, { useState } from 'react';
import { useTranslation } from '@/i18n/provider';
import guestService from '@/services/guest';

interface DeleteGuestDataButtonProps {
  orderReference: string;
  email?: string;
  phone?: string;
  onSuccess: () => void;
}

export default function DeleteGuestDataButton({
  orderReference,
  email,
  phone,
  onSuccess,
}: DeleteGuestDataButtonProps) {
  const { t } = useTranslation(['guest_data', 'common']);
  const [showModal, setShowModal] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState('');

  const handleDelete = async () => {
    setIsDeleting(true);
    setError('');

    try {
      await guestService.deleteGuestOrderData(orderReference, {
        email: email || null,
        phone: phone || null,
      });

      // Success - notify parent
      onSuccess();
    } catch (err: any) {
      switch (err.response?.status) {
        case 400:
          setError(t('guest_data:errors.not_completed_or_cancelled'));
          break;
        case 403:
          setError(t('guest_data:errors.verification_failed'));
          break;
        case 404:
          setError(t('guest_data:errors.order_not_found'));
          break;
        case 409:
          setError(t('guest_data:errors.already_deleted'));
          break;
        default:
          setError(err.response?.data?.error || t('guest_data:errors.deletion_failed'));
          break;
      }
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <>
      <button
        onClick={() => setShowModal(true)}
        className="w-full bg-red-600 text-white py-3 px-4 rounded-lg font-medium hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 transition-colors"
      >
        {t('guest_data:actions.delete_data')}
      </button>

      {/* Confirmation Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 overflow-y-auto" aria-labelledby="modal-title" role="dialog" aria-modal="true">
          <div className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
            {/* Background overlay */}
            <div
              className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
              aria-hidden="true"
              onClick={() => !isDeleting && setShowModal(false)}
            ></div>

            {/* Center modal */}
            <span className="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">&#8203;</span>

            <div className="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
              <div className="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                <div className="sm:flex sm:items-start">
                  <div className="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-red-100 sm:mx-0 sm:h-10 sm:w-10">
                    <svg className="h-6 w-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                  </div>
                  <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
                    <h3 className="text-lg leading-6 font-medium text-gray-900" id="modal-title">
                      {t('guest_data:deletion.confirm_title')}
                    </h3>
                    <div className="mt-2">
                      <p className="text-sm text-gray-500 mb-4">
                        {t('guest_data:deletion.confirm_message')}
                      </p>

                      {/* What will be deleted */}
                      <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-4">
                        <p className="text-sm font-medium text-red-800 mb-2">
                          {t('guest_data:deletion.will_be_deleted')}
                        </p>
                        <ul className="text-sm text-red-700 list-disc list-inside space-y-1">
                          <li>{t('guest_data:deletion.deleted_items.name')}</li>
                          <li>{t('guest_data:deletion.deleted_items.email')}</li>
                          <li>{t('guest_data:deletion.deleted_items.phone')}</li>
                          <li>{t('guest_data:deletion.deleted_items.address')}</li>
                          <li>{t('guest_data:deletion.deleted_items.ip_address')}</li>
                        </ul>
                      </div>

                      {/* What will be preserved */}
                      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
                        <p className="text-sm font-medium text-blue-800 mb-2">
                          {t('guest_data:deletion.will_be_preserved')}
                        </p>
                        <ul className="text-sm text-blue-700 list-disc list-inside space-y-1">
                          <li>{t('guest_data:deletion.preserved_items.order_reference')}</li>
                          <li>{t('guest_data:deletion.preserved_items.order_items')}</li>
                          <li>{t('guest_data:deletion.preserved_items.amounts')}</li>
                          <li>{t('guest_data:deletion.preserved_items.status')}</li>
                        </ul>
                      </div>

                      <p className="text-xs text-gray-500">
                        {t('guest_data:deletion.irreversible_warning')}
                      </p>
                    </div>
                  </div>
                </div>

                {/* Error message */}
                {error && (
                  <div className="mt-4 bg-red-50 border border-red-200 rounded-lg p-4">
                    <p className="text-sm text-red-700">{error}</p>
                  </div>
                )}
              </div>

              <div className="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
                <button
                  type="button"
                  disabled={isDeleting}
                  onClick={handleDelete}
                  className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-red-600 text-base font-medium text-white hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:ml-3 sm:w-auto sm:text-sm disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isDeleting ? t('common:loading') : t('guest_data:actions.confirm_delete')}
                </button>
                <button
                  type="button"
                  disabled={isDeleting}
                  onClick={() => setShowModal(false)}
                  className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t('common:cancel')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
