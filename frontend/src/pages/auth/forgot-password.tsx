import { useTranslation } from 'react-i18next';
import PublicLayout from '../../components/layout/PublicLayout';
import PasswordResetRequestForm from '../../components/forms/PasswordResetRequestForm';

export default function ForgotPasswordPage() {
  const { t } = useTranslation('auth');

  return (
    <PublicLayout>
      <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-md w-full space-y-8">
          <div>
            <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
              {t('forgotPassword.title')}
            </h2>
          </div>
          <div className="bg-white p-8 rounded-lg shadow">
            <PasswordResetRequestForm />
          </div>
        </div>
      </div>
    </PublicLayout>
  );
}
