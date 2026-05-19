import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import PricingSection from '@/components/landing/PricingSection';

describe('PricingSection', () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    global.fetch = originalFetch;
    jest.restoreAllMocks();
  });

  it('uses fallback pricing when public plans API fails', async () => {
    global.fetch = jest.fn().mockRejectedValue(new Error('network error'));

    render(<PricingSection />);

    expect(screen.getByText(/Rp\s*299\.000/)).toBeInTheDocument();
    expect(screen.getByText('7-day free trial')).toBeInTheDocument();

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/api/v1/public/plans');
    });
  });

  it('renders pricing returned by public plans API', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        monthly_price_idr: 400000,
        annual_discount_pct: 30,
        trial_days: 14,
      }),
    });

    render(<PricingSection />);

    await waitFor(() => {
      expect(screen.getByText(/Rp\s*400\.000/)).toBeInTheDocument();
    });

    expect(screen.getByText('14-day free trial')).toBeInTheDocument();
    expect(screen.getByText('Save 30%')).toBeInTheDocument();
  });
});
