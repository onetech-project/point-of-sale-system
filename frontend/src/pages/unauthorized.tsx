import { useRouter } from 'next/router';
import { useTranslation } from 'react-i18next';
import Button from '../components/ui/Button';
import PublicLayout from '../components/layout/PublicLayout';

export default function UnauthorizedPage() {
  const { t } = useTranslation('common');
  const router = useRouter();

  return (
    <PublicLayout>
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full text-center p-8">
          <div className="bg-white rounded-lg shadow-lg p-8">
            <div className="mb-6">
              <svg
                className="mx-auto h-16 w-16 text-red-500"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>
            <h1 className="text-3xl font-bold text-gray-900 mb-4">
              {t('unauthorized.title')}
            </h1>
            <p className="text-gray-600 mb-8">
              {t('unauthorized.message')}
            </p>
            <div className="space-x-4">
              <Button
                onClick={() => router.back()}
                variant="outline"
              >
                {t('unauthorized.goBack')}
              </Button>
              <Button
                onClick={() => router.push('/dashboard')}
              >
                {t('unauthorized.goHome')}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </PublicLayout>
  );
}
