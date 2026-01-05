-- Migration 000034: Seed privacy_policy v1.0.0
-- Purpose: Initial Indonesian privacy policy text per UU PDP No.27 Tahun 2022
-- Feature: 006-uu-pdp-compliance

INSERT INTO privacy_policies (version, policy_text_id, policy_text_en, effective_date, is_major_update, is_current)
VALUES (
    '1.0.0',
    -- Indonesian policy text (legally binding)
    'KEBIJAKAN PRIVASI - Sistem Point of Sale (POS)

Terakhir diperbarui: 2 Januari 2026

Kebijakan Privasi ini menjelaskan bagaimana kami mengumpulkan, menggunakan, dan melindungi data pribadi Anda sesuai dengan Undang-Undang Pelindungan Data Pribadi No. 27 Tahun 2022 (UU PDP).

1. DATA YANG KAMI KUMPULKAN

Kami mengumpulkan kategori data pribadi berikut:
- Informasi Akun: Alamat email, nama depan, nama belakang, nomor telepon
- Informasi Pesanan: Nama pelanggan, nomor telepon, alamat email, alamat pengiriman, koordinat geografis
- Data Teknis: Alamat IP, informasi perangkat, riwayat login, token sesi
- Data Pembayaran: Token pembayaran Midtrans (kami tidak menyimpan nomor kartu lengkap)
- Cookie dan Teknologi Pelacakan: Cookie sesi, cookie analitik (dengan persetujuan)

2. TUJUAN PEMROSESAN DATA

Kami memproses data pribadi Anda untuk:
a) Operasional (Wajib): Manajemen akun, pemrosesan pesanan, akses katalog produk, otentikasi pengguna
b) Analitik (Opsional): Analisis pola penggunaan, optimasi kinerja, pemahaman perilaku pengguna
c) Pemasaran (Opsional): Komunikasi promosi, penawaran personal, buletin
d) Pembayaran (Wajib): Pemrosesan pembayaran melalui Midtrans (pihak ketiga)

3. DASAR HUKUM PEMROSESAN

Kami memproses data pribadi Anda berdasarkan:
- Persetujuan eksplisit Anda (Pasal 20 UU PDP) - dikumpulkan saat pendaftaran dan checkout
- Pelaksanaan kontrak layanan
- Kewajiban hukum (retensi data pajak, catatan audit)

4. PERIODE RETENSI DATA

- Akun Aktif: Data disimpan selama akun aktif
- Akun yang Dihapus: 90 hari masa tenggang sebelum penghapusan permanen
- Pesanan Tamu: 5 tahun (sesuai persyaratan pajak Indonesia)
- Catatan Audit: 7 tahun (standar kepatuhan Indonesia)
- Data Sementara: 48 jam (token verifikasi, token reset kata sandi)

5. BERBAGI DATA DENGAN PIHAK KETIGA

Kami membagikan data Anda dengan:
- Midtrans (Processor Pembayaran): Memproses transaksi pembayaran
  Kebijakan Privasi: https://midtrans.com/privacy-policy
  Lokasi Data: Indonesia
  Dasar Hukum: Persetujuan eksplisit untuk pemrosesan pembayaran

Kami TIDAK menjual data pribadi Anda kepada pihak ketiga.

6. LANGKAH-LANGKAH KEAMANAN

Kami menerapkan langkah-langkah keamanan berikut:
- Enkripsi at Rest: Semua data pribadi dienkripsi di penyimpanan
- Kontrol Akses: Isolasi multi-tenant, kontrol akses berbasis peran (RBAC)
- Logging Audit: Jejak audit yang tidak dapat diubah untuk semua akses data
- Masking Log: Data sensitif disamarkan di log aplikasi

7. HAK ANDA (Pasal 3-6 UU PDP)

Anda memiliki hak untuk:
a) Akses: Lihat semua data pribadi yang kami miliki tentang Anda
b) Koreksi: Perbarui informasi yang tidak akurat atau tidak lengkap
c) Penghapusan: Minta penghapusan data pribadi Anda (dengan pengecualian)
d) Pencabutan Persetujuan: Cabut persetujuan opsional kapan saja
e) Portabilitas Data: Ekspor data Anda dalam format JSON
f) Pengaduan: Ajukan pengaduan kepada otoritas pelindungan data

Untuk menggunakan hak Anda, hubungi: privacy@pos-system.id

8. BATASAN PENGHAPUSAN DATA

Kami tidak dapat menghapus data jika:
- Diperlukan untuk kewajiban hukum (catatan pajak, catatan audit)
- Diperlukan untuk klaim hukum atau penyelesaian sengketa
- Diperlukan untuk keamanan dan pencegahan penipuan

9. PROSES PENGADUAN

Jika Anda memiliki keluhan tentang pemrosesan data kami:
1. Hubungi kami di privacy@pos-system.id
2. Kami akan merespons dalam 14 hari kalender (Pasal 6 UU PDP)
3. Jika tidak puas, ajukan pengaduan ke Kementerian Komunikasi dan Informatika

10. PERUBAHAN PADA KEBIJAKAN INI

Kami dapat memperbarui kebijakan ini dari waktu ke waktu:
- Pembaruan Minor (v1.x.x): Notifikasi non-blocking
- Pembaruan Mayor (v2.0.0): Persetujuan ulang diperlukan

11. INFORMASI KONTAK

Email: privacy@pos-system.id
Waktu Respons: 14 hari kalender
Alamat: [Alamat perusahaan akan ditambahkan]

Dengan menggunakan layanan kami, Anda mengakui bahwa Anda telah membaca dan memahami Kebijakan Privasi ini.',

-- English translation (optional)
'PRIVACY POLICY - Point of Sale (POS) System

Last updated: January 2, 2026

This Privacy Policy explains how we collect, use, and protect your personal data in accordance with Indonesian Personal Data Protection Law No. 27 of 2022 (UU PDP).

1. DATA WE COLLECT
- Account Information: Email, first name, last name, phone number
- Order Information: Customer name, phone, email, delivery address, geographic coordinates
- Technical Data: IP addresses, device information, login history, session tokens
- Payment Data: Midtrans payment tokens (we do not store full card numbers)
- Cookies and Tracking: Session cookies, analytics cookies (with consent)

2. PROCESSING PURPOSES
a) Operational (Required): Account management, order processing, product catalog access
b) Analytics (Optional): Usage analysis, performance optimization
c) Marketing (Optional): Promotional communications, personalized offers
d) Payment (Required): Payment processing via Midtrans (third party)

3. LEGAL BASIS
- Explicit consent (Article 20 UU PDP)
- Contract performance
- Legal obligations

4. RETENTION PERIODS
- Active Accounts: While account is active
- Deleted Accounts: 90-day grace period
- Guest Orders: 5 years (Indonesian tax requirements)
- Audit Logs: 7 years (Indonesian compliance standards)
- Temporary Data: 48 hours

5. THIRD-PARTY SHARING
- Midtrans (Payment Processor): For payment transactions
  Privacy Policy: https://midtrans.com/privacy-policy

We do NOT sell your personal data.

6. SECURITY MEASURES
- Encryption at Rest
- Access Controls (multi-tenant isolation, RBAC)
- Audit Logging
- Log Masking

7. YOUR RIGHTS (Articles 3-6 UU PDP)
- Access, Correction, Deletion, Consent Withdrawal, Data Portability, Complaint

Contact: privacy@pos-system.id

8. DELETION LIMITATIONS
Cannot delete data required for legal obligations or security.

9. COMPLAINT PROCESS
Contact privacy@pos-system.id - Response within 14 calendar days.

10. POLICY CHANGES
Minor updates: Non-blocking notification
Major updates: Re-consent required

11. CONTACT
Email: privacy@pos-system.id
Response Time: 14 calendar days',
    
    '2026-01-02 00:00:00+00',
    TRUE,  -- is_major_update (initial version)
    TRUE   -- is_current
)
ON CONFLICT (version) DO NOTHING;