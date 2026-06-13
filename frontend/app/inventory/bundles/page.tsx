import BundleManager from '@/components/inventory/BundleManager';
import InventoryShell from '@/components/inventory/InventoryShell';

export default function InventoryBundlesPage() {
  return (
    <InventoryShell
      title="Bundles"
      subtitle="Create product bundles and preview their expanded recipe costs."
    >
      <BundleManager />
    </InventoryShell>
  );
}
