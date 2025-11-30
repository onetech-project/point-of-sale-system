import { useRouter } from 'next/router';
import { useTranslation } from 'react-i18next';
import PublicLayout from '../../components/layout/PublicLayout';
import PasswordResetForm from '../../components/forms/PasswordResetForm';

export default function ResetPasswordPage() {
  const { t } = useTranslation('auth');
  const router = useRouter();
  const { token } = router.query;

  if (!token || typeof token !== 'string') {
    return (
      <PublicLayout>
        <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
          <div className="max-w-md w-full">
            <div className="bg-white p-8 rounded-lg shadow text-center">
              <h2 className="text-2xl font-bold text-red-600 mb-4">
                {t('resetPassword.invalidLink')}
              </h2>
              <p className="text-gray-600">
                {t('resetPassword.invalidLinkMessage')}
              </p>
            </div>
          </div>
        </div>
      </PublicLayout>
    );
  }

  return (
    <PublicLayout>
      <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-md w-full space-y-8">
          <div>
            <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
              {t('resetPassword.title')}
            </h2>
          </div>
          <div className="bg-white p-8 rounded-lg shadow">
            <PasswordResetForm token={token} />
          </div>
        </div>
      </div>
    </PublicLayout>
  );
}
