import { redirect } from 'next/navigation';

interface OfflineOrderDetailPageProps {
  params: Promise<{
    id: string;
  }>;
}

export default async function OfflineOrderDetailPage({ params }: OfflineOrderDetailPageProps) {
  const { id } = await params;

  redirect(`/orders?order_id=${encodeURIComponent(id)}&order_type=offline`);
}
