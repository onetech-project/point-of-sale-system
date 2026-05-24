import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { MetricCard } from './MetricCard';

it('uses responsive and wrapping classes for large KPI values', () => {
  render(<MetricCard title="Revenue" value="Rp123.456.789.000,00" />);

  const value = screen.getByText('Rp123.456.789.000,00');
  expect(value).toHaveClass('text-xl');
  expect(value).toHaveClass('lg:text-2xl');
  expect(value).toHaveClass('xl:text-3xl');
  expect(value).toHaveClass('leading-tight');
  expect(value).toHaveClass('break-words');
  expect(value).toHaveClass('[overflow-wrap:anywhere]');
});
