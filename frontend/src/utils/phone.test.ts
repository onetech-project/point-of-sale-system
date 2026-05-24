import { sanitizeNumericPhone } from './phone';

describe('sanitizeNumericPhone', () => {
  it('keeps only ASCII digits', () => {
    expect(sanitizeNumericPhone('+62 812-3456 abc')).toBe('628123456');
  });
});
