-- Migration 000047: Add guest checkout consent purposes
-- Purpose: order_processing, order_communications, promotional_communications, payment_processing_midtrans
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
        'order_processing',
        'Order Processing',
        'Pemrosesan Pesanan',
        'Process your order information to fulfill your purchase, manage inventory, and provide customer support',
        'Memproses informasi pesanan Anda untuk memenuhi pembelian, mengelola inventaris, dan memberikan dukungan pelanggan',
        TRUE,
        5
    ),
    (
        'order_communications',
        'Order Communications',
        'Komunikasi Pesanan',
        'Send order confirmations, shipping updates, and delivery notifications via email or SMS',
        'Mengirim konfirmasi pesanan, pembaruan pengiriman, dan notifikasi pengiriman melalui email atau SMS',
        FALSE,
        6
    ),
    (
        'promotional_communications',
        'Promotional Communications',
        'Komunikasi Promosi',
        'Send promotional offers, discount codes, and marketing messages related to your orders',
        'Mengirim penawaran promosi, kode diskon, dan pesan pemasaran terkait pesanan Anda',
        FALSE,
        7
    ),
    (
        'payment_processing_midtrans',
        'Payment Processing (Midtrans)',
        'Pemrosesan Pembayaran (Midtrans)',
        'Share payment information with Midtrans payment gateway to securely process your transaction',
        'Membagikan informasi pembayaran dengan gateway pembayaran Midtrans untuk memproses transaksi Anda dengan aman',
        TRUE,
        8
    )
ON CONFLICT (purpose_code) DO NOTHING;

-- Comments
COMMENT ON TABLE consent_purposes IS 'Added 4 guest checkout purposes: order_processing (required), order_communications (optional), promotional_communications (optional), payment_processing_midtrans (required)';