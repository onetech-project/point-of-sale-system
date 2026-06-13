'use client';

import InventoryOverview from '@/components/inventory/InventoryOverview';
import InventoryShell from '@/components/inventory/InventoryShell';

export default function InventoryPage() {
  return (
    <InventoryShell
      title="Inventory"
      subtitle="Ingredient stock, units, valuation, and movement controls."
    >
      <InventoryOverview />
    </InventoryShell>
  );
}
