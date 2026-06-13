'use client';

import InventoryShell from '@/components/inventory/InventoryShell';
import StockManager from '@/components/inventory/StockManager';

export default function InventoryStockPage() {
  return (
    <InventoryShell
      title="Stock"
      subtitle="Record stock movements and review ingredient inventory reports."
    >
      <StockManager />
    </InventoryShell>
  );
}
