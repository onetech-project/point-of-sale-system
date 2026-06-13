import { render, screen } from '@testing-library/react';
import AnalyticsPage from './page';
import InventoryAnalyticsPage from './inventory/page';

// Mock the analytics service
jest.mock('@/services/analytics', () => ({
  __esModule: true,
  default: {
    getSalesOverview: jest.fn().mockResolvedValue({
      metrics: {
        total_revenue: 5000000,
        total_orders: 100,
        average_order_value: 50000,
        inventory_value: 2000000,
        revenue_change: 10.5,
        orders_change: 5.2,
        aov_change: 3.1,
        offline_order_count: 30,
        offline_revenue: 1500000,
        offline_percentage: 30,
        online_order_count: 70,
        online_revenue: 3500000,
        installment_count: 10,
        installment_revenue: 500000,
        pending_installments: 200000,
      },
      sales_chart: [],
      top_products: [],
      category_breakdown: [],
    }),
    getTopProducts: jest.fn().mockResolvedValue({
      top_by_revenue: [
        {
          product_id: 1,
          name: 'Test Product',
          quantity_sold: 50,
          revenue: 2500000,
          sku: 'TEST-001',
        },
      ],
      top_by_quantity: [],
      bottom_by_revenue: [],
      bottom_by_quantity: [],
    }),
    getProductProfitability: jest.fn().mockResolvedValue({
      products: [
        {
          product_id: '1',
          product_name: 'Profitable Product',
          total_revenue: 1000000,
          total_cogs: 600000,
          gross_profit: 400000,
          gross_margin_pct: 40,
          total_quantity_sold: 20,
          avg_selling_price: 50000,
          avg_cogs_per_unit: 30000,
          recipe_version: 1,
        },
      ],
      start_date: '2026-05-01',
      end_date: '2026-06-01',
      total_items: 1,
    }),
    getIngredientForecast: jest.fn().mockResolvedValue({
      ingredients: [
        {
          ingredient_id: '1',
          ingredient_name: 'Flour',
          total_consumed: 100,
          base_unit: 'kg',
          total_cost: 500000,
          avg_cost_per_unit: 5000,
          current_stock: 50,
          stock_after_forecast: 0,
          days_until_stockout: 5,
        },
      ],
      start_date: '2026-05-01',
      end_date: '2026-06-01',
      daily_burn_rate: [],
    }),
    simulateRevenueTarget: jest.fn().mockResolvedValue({
      target: {
        target_revenue: 10000000,
        period_days: 30,
        daily_target: 333333,
        weekly_target: 2333333,
      },
      recommended_mix: [],
      estimated_ingredient_usage: [],
      estimated_budget: 5000000,
      projected_gross_profit: 3000000,
      projected_gross_margin: 30,
      feasibility_score: 85,
      feasibility_notes: ['Revenue target appears feasible'],
    }),
    simulateBudget: jest.fn().mockResolvedValue({
      budget: 5000000,
      start_date: '2026-05-01',
      end_date: '2026-06-01',
      recommended_mix: [],
      ingredient_breakdown: [],
      projected_revenue: 8000000,
      projected_gross_profit: 2400000,
      projected_gross_margin: 30,
      budget_utilization: 100,
    }),
    getProductRecommendations: jest.fn().mockResolvedValue({
      recommendations: [
        {
          product_id: '1',
          product_name: 'Recommended Product',
          total_score: 0.85,
          margin_score: 0.35,
          velocity_score: 0.25,
          stock_readiness_score: 0.25,
          trend_score: 0.15,
          bundle_opportunity: true,
          recommendation_note: 'high margin champion',
        },
      ],
      generated_at: '2026-06-13T00:00:00Z',
      scoring_weights: {
        margin: 0.35,
        velocity: 0.25,
        stock_readiness: 0.25,
        trend: 0.15,
      },
    }),
  },
}));

// Mock ProtectedRoute
jest.mock('@/components/auth/ProtectedRoute', () => {
  return function MockProtectedRoute({ children }: { children: React.ReactNode }) {
    return <div>{children}</div>;
  };
});

// Mock DashboardLayout
jest.mock('@/components/layout/DashboardLayout', () => {
  return function MockDashboardLayout({ children }: { children: React.ReactNode }) {
    return <div>{children}</div>;
  };
});

describe('Analytics Pages', () => {
  describe('AnalyticsPage', () => {
    it('renders the analytics page with metrics', async () => {
      render(<AnalyticsPage />);

      // Wait for data to load
      await screen.findByText('Total Revenue');

      expect(screen.getByText('Total Revenue')).toBeInTheDocument();
      expect(screen.getByText('Total Orders')).toBeInTheDocument();
      expect(screen.getByText('Average Order Value')).toBeInTheDocument();
      expect(screen.getByText('Inventory Value')).toBeInTheDocument();
    });

    it('displays revenue metrics correctly', async () => {
      render(<AnalyticsPage />);

      await screen.findByText('Total Revenue');

      expect(screen.getByText('Rp 5,000,000')).toBeInTheDocument();
      expect(screen.getByText('100')).toBeInTheDocument();
      expect(screen.getByText('Rp 50,000')).toBeInTheDocument();
    });

    it('shows change percentages', async () => {
      render(<AnalyticsPage />);

      await screen.findByText('+10.5% vs last period');

      expect(screen.getByText('+10.5% vs last period')).toBeInTheDocument();
      expect(screen.getByText('+5.2% vs last period')).toBeInTheDocument();
    });
  });

  describe('InventoryAnalyticsPage', () => {
    it('renders the inventory analytics page with tabs', async () => {
      render(<InventoryAnalyticsPage />);

      expect(screen.getByText('Inventory Analytics')).toBeInTheDocument();
      expect(screen.getByText('Profitability')).toBeInTheDocument();
      expect(screen.getByText('Ingredient Forecast')).toBeInTheDocument();
      expect(screen.getByText('Revenue Simulator')).toBeInTheDocument();
      expect(screen.getByText('Budget Simulator')).toBeInTheDocument();
      expect(screen.getByText('Recommendations')).toBeInTheDocument();
    });

    it('displays date range picker', async () => {
      render(<InventoryAnalyticsPage />);

      expect(screen.getByDisplayValue('2026-05-13')).toBeInTheDocument();
      expect(screen.getByDisplayValue('2026-06-13')).toBeInTheDocument();
    });

    it('shows profitability tab by default', async () => {
      render(<InventoryAnalyticsPage />);

      await screen.findByText('Profitable Product');

      expect(screen.getByText('Profitable Product')).toBeInTheDocument();
      expect(screen.getAllByText('Rp 1,000,000').length).toBeGreaterThan(0);
      expect(screen.getAllByText('Rp 400,000').length).toBeGreaterThan(0);
    });
  });
});
