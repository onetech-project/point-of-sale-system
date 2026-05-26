import { redirect } from 'next/navigation';

export default function NewOfflineOrderPage() {
  redirect('/orders?mode=new-offline');
}
