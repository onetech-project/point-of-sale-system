import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { AuditLog } from './AuditLog';
import auditService from '../../services/audit';

jest.mock('@/i18n/provider', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

jest.mock('../../services/audit', () => ({
  __esModule: true,
  default: {
    getTenantAuditEvents: jest.fn(),
  },
}));

const mockAuditService = auditService as jest.Mocked<typeof auditService>;

it('includes team audit filters', async () => {
  mockAuditService.getTenantAuditEvents.mockResolvedValue({
    events: [],
    pagination: {
      total: 0,
      limit: 100,
      offset: 0,
    },
  });

  render(<AuditLog />);

  expect(screen.getByRole('option', { name: 'ACCESS' })).toBeInTheDocument();
  expect(screen.getByRole('option', { name: 'Invitation' })).toBeInTheDocument();
  await waitFor(() => {
    expect(mockAuditService.getTenantAuditEvents).toHaveBeenCalled();
  });
});
