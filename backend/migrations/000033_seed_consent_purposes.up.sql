-- Migration 000033: Seed consent_purposes with initial purposes
-- Purpose: Operational, analytics, advertising, and third-party payment processing
-- Feature: 006-uu-pdp-compliance (UU PDP No.27 Tahun 2022)

INSERT INTO
    consent_purposes (
        purpose_code,
        purpose_name_en,
        purpose_name_id,
        description_en,
        description_id,
        is_required,
        display_order
    )
VALUES (
        'operational',
        'Operational Data Processing',
        'Pemrosesan Data Operasional',
        'Process your data to provide core services: account management, order processing, product catalog access',
        'Memproses data Anda untuk menyediakan layanan inti: manajemen akun, pemrosesan pesanan, akses katalog produk',
        TRUE,
        1
    ),
    (
        'analytics',
        'Analytics and Performance',
        'Analitik dan Kinerja',
        'Analyze usage patterns to improve service quality, optimize performance, and understand user behavior',
        'Menganalisis pola penggunaan untuk meningkatkan kualitas layanan, mengoptimalkan kinerja, dan memahami perilaku pengguna',
        FALSE,
        2
    ),
    (
        'advertising',
        'Advertising and Marketing',
        'Iklan dan Pemasaran',
        'Send promotional communications, personalized offers, and marketing messages',
        'Mengirim komunikasi promosi, penawaran personal, dan pesan pemasaran',
        FALSE,
        3
    ),
    (
        'third_party_midtrans',
        'Third-Party Payment Processing (Midtrans)',
        'Pemrosesan Pembayaran Pihak Ketiga (Midtrans)',
        'Share payment information with Midtrans payment gateway to process transactions securely',
        'Membagikan informasi pembayaran dengan gateway pembayaran Midtrans untuk memproses transaksi dengan aman',
        TRUE,
        4
    )
ON CONFLICT (purpose_code) DO NOTHING;

-- Comments
COMMENT ON TABLE consent_purposes IS 'Seeded with 4 default purposes: operational (required), analytics (optional), advertising (optional), third_party_midtrans (required)';