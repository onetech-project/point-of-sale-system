import '@testing-library/jest-dom'

// Lightweight i18n shim for tests to avoid initializing full i18next instance
jest.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (s) => s,
    i18n: { changeLanguage: () => Promise.resolve() },
  }),
  Trans: ({ children }) => children,
}))
