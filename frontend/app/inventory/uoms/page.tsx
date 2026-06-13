'use client';

import InventoryShell from '@/components/inventory/InventoryShell';
import UOMManager from '@/components/inventory/UOMManager';

export default function InventoryUOMsPage() {
  return (
    <InventoryShell title="Units" subtitle="Manage inventory units of measure.">
      <UOMManager />
    </InventoryShell>
  );
}
