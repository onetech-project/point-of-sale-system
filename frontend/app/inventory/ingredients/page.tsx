'use client';

import IngredientManager from '@/components/inventory/IngredientManager';
import InventoryShell from '@/components/inventory/InventoryShell';

export default function InventoryIngredientsPage() {
  return (
    <InventoryShell
      title="Ingredients"
      subtitle="Create ingredients, set base units, maintain thresholds, and manage conversions."
    >
      <IngredientManager />
    </InventoryShell>
  );
}
