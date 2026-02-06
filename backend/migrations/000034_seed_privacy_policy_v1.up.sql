-- Migration 000034: Seed privacy_policy v1.0.0
-- Purpose: Initial Indonesian privacy policy text per UU PDP No.27 Tahun 2022
-- Feature: 006-uu-pdp-compliance

INSERT INTO privacy_policies (version, policy_text_id, policy_text_en, effective_date, is_major_update, is_current)
VALUES (
    '1.0.0',
    -- Indonesian policy text (legally binding) - HTML formatted
    '<div class="privacy-policy">
<section class="introduction">
<p class="last-updated">Terakhir diperbarui: 2 Januari 2026</p>
<p>Kebijakan Privasi ini menjelaskan bagaimana kami mengumpulkan, menggunakan, dan melindungi data pribadi Anda sesuai dengan <strong>Undang-Undang Pelindungan Data Pribadi No. 27 Tahun 2022 (UU PDP)</strong>.</p>
</section>

<section class="data-collection">
<h2>1. DATA YANG KAMI KUMPULKAN</h2>
<p>Kami mengumpulkan kategori data pribadi berikut:</p>
<ul>
<li><strong>Informasi Akun:</strong> Alamat email, nama depan, nama belakang, nomor telepon</li>
<li><strong>Informasi Pesanan:</strong> Nama pelanggan, nomor telepon, alamat email, alamat pengiriman, koordinat geografis</li>
<li><strong>Data Teknis:</strong> Alamat IP, informasi perangkat, riwayat login, token sesi</li>
<li><strong>Data Pembayaran:</strong> Token pembayaran Midtrans (kami tidak menyimpan nomor kartu lengkap)</li>
<li><strong>Cookie dan Teknologi Pelacakan:</strong> Cookie sesi, cookie analitik (dengan persetujuan)</li>
</ul>
</section>

<section class="processing-purposes">
<h2>2. TUJUAN PEMROSESAN DATA</h2>
<p>Kami memproses data pribadi Anda untuk:</p>
<ol type="a">
<li><strong>Operasional (Wajib):</strong> Manajemen akun, pemrosesan pesanan, akses katalog produk, otentikasi pengguna</li>
<li><strong>Analitik (Opsional):</strong> Analisis pola penggunaan, optimasi kinerja, pemahaman perilaku pengguna</li>
<li><strong>Pemasaran (Opsional):</strong> Komunikasi promosi, penawaran personal, buletin</li>
<li><strong>Pembayaran (Wajib):</strong> Pemrosesan pembayaran melalui Midtrans (pihak ketiga)</li>
</ol>
</section>

<section class="legal-basis">
<h2>3. DASAR HUKUM PEMROSESAN</h2>
<p>Kami memproses data pribadi Anda berdasarkan:</p>
<ul>
<li>Persetujuan eksplisit Anda (Pasal 20 UU PDP) - dikumpulkan saat pendaftaran dan checkout</li>
<li>Pelaksanaan kontrak layanan</li>
<li>Kewajiban hukum (retensi data pajak, catatan audit)</li>
</ul>
</section>

<section class="retention">
<h2>4. PERIODE RETENSI DATA</h2>
<ul>
<li><strong>Akun Aktif:</strong> Data disimpan selama akun aktif</li>
<li><strong>Akun yang Dihapus:</strong> 90 hari masa tenggang sebelum penghapusan permanen</li>
<li><strong>Pesanan Tamu:</strong> 5 tahun (sesuai persyaratan pajak Indonesia)</li>
<li><strong>Catatan Audit:</strong> 7 tahun (standar kepatuhan Indonesia)</li>
<li><strong>Data Sementara:</strong> 48 jam (token verifikasi, token reset kata sandi)</li>
</ul>
</section>

<section class="third-party">
<h2>5. BERBAGI DATA DENGAN PIHAK KETIGA</h2>
<p>Kami membagikan data Anda dengan:</p>
<div class="third-party-box">
<h4>Midtrans (Processor Pembayaran)</h4>
<p>Memproses transaksi pembayaran</p>
<ul>
<li><strong>Kebijakan Privasi:</strong> <a href="https://midtrans.com/id/pemberitahuan-privasi" target="_blank" rel="noopener noreferrer">https://midtrans.com/id/pemberitahuan-privasi</a></li>
<li><strong>Lokasi Data:</strong> Indonesia</li>
<li><strong>Dasar Hukum:</strong> Persetujuan eksplisit untuk pemrosesan pembayaran</li>
</ul>
</div>
<p class="highlight"><strong>Kami TIDAK menjual data pribadi Anda kepada pihak ketiga.</strong></p>
</section>

<section class="security">
<h2>6. LANGKAH-LANGKAH KEAMANAN</h2>
<p>Kami menerapkan langkah-langkah keamanan berikut:</p>
<ul>
<li><strong>Enkripsi at Rest:</strong> Semua data pribadi dienkripsi di penyimpanan</li>
<li><strong>Kontrol Akses:</strong> Isolasi multi-tenant, kontrol akses berbasis peran (RBAC)</li>
<li><strong>Logging Audit:</strong> Jejak audit yang tidak dapat diubah untuk semua akses data</li>
<li><strong>Masking Log:</strong> Data sensitif disamarkan di log aplikasi</li>
</ul>
</section>

<section class="your-rights">
<h2>7. HAK ANDA (Pasal 3-6 UU PDP)</h2>
<p>Anda memiliki hak untuk:</p>
<ol type="a">
<li><strong>Akses:</strong> Lihat semua data pribadi yang kami miliki tentang Anda</li>
<li><strong>Koreksi:</strong> Perbarui informasi yang tidak akurat atau tidak lengkap</li>
<li><strong>Penghapusan:</strong> Minta penghapusan data pribadi Anda (dengan pengecualian)</li>
<li><strong>Pencabutan Persetujuan:</strong> Cabut persetujuan opsional kapan saja</li>
<li><strong>Portabilitas Data:</strong> Ekspor data Anda dalam format JSON</li>
<li><strong>Pengaduan:</strong> Ajukan pengaduan kepada otoritas pelindungan data</li>
</ol>
<p>Untuk menggunakan hak Anda, hubungi: <a href="mailto:posku-service@proton.me">posku-service@proton.me</a></p>
</section>

<section class="deletion-limits">
<h2>8. BATASAN PENGHAPUSAN DATA</h2>
<p>Kami tidak dapat menghapus data jika:</p>
<ul>
<li>Diperlukan untuk kewajiban hukum (catatan pajak, catatan audit)</li>
<li>Diperlukan untuk klaim hukum atau penyelesaian sengketa</li>
<li>Diperlukan untuk keamanan dan pencegahan penipuan</li>
</ul>
</section>

<section class="complaint-process">
<h2>9. PROSES PENGADUAN</h2>
<p>Jika Anda memiliki keluhan tentang pemrosesan data kami:</p>
<ol>
<li>Hubungi kami di <a href="mailto:posku-service@proton.me">posku-service@proton.me</a></li>
<li>Kami akan merespons dalam 14 hari kalender (Pasal 6 UU PDP)</li>
<li>Jika tidak puas, ajukan pengaduan ke Kementerian Komunikasi dan Informatika</li>
</ol>
</section>

<section class="policy-changes">
<h2>10. PERUBAHAN PADA KEBIJAKAN INI</h2>
<p>Kami dapat memperbarui kebijakan ini dari waktu ke waktu:</p>
<ul>
<li><strong>Pembaruan Minor (v1.x.x):</strong> Notifikasi non-blocking</li>
<li><strong>Pembaruan Mayor (v2.0.0):</strong> Persetujuan ulang diperlukan</li>
</ul>
</section>

<section class="contact">
<h2>11. INFORMASI KONTAK</h2>
<ul>
<li><strong>Email:</strong> <a href="mailto:posku-service@proton.me">posku-service@proton.me</a></li>
<li><strong>Waktu Respons:</strong> 14 hari kalender</li>
<li><strong>Alamat:</strong> - </li>
</ul>
</section>

<section class="acknowledgment">
<p class="disclaimer">Dengan menggunakan layanan kami, Anda mengakui bahwa Anda telah membaca dan memahami Kebijakan Privasi ini.</p>
</section>
</div>',

-- English translation (optional) - HTML formatted


'<div class="privacy-policy">
<section class="introduction">
<p class="last-updated">Last updated: January 2, 2026</p>
<p>This Privacy Policy explains how we collect, use, and protect your personal data in accordance with <strong>Indonesian Personal Data Protection Law No. 27 of 2022 (UU PDP)</strong>.</p>
</section>

<section class="data-collection">
<h2>1. DATA WE COLLECT</h2>
<p>We collect the following categories of personal data:</p>
<ul>
<li><strong>Account Information:</strong> Email address, first name, last name, phone number</li>
<li><strong>Order Information:</strong> Customer name, phone number, email address, delivery address, geographic coordinates</li>
<li><strong>Technical Data:</strong> IP addresses, device information, login history, session tokens</li>
<li><strong>Payment Data:</strong> Midtrans payment tokens (we do not store full card numbers)</li>
<li><strong>Cookies and Tracking:</strong> Session cookies, analytics cookies (with consent)</li>
</ul>
</section>

<section class="processing-purposes">
<h2>2. PROCESSING PURPOSES</h2>
<p>We process your personal data for:</p>
<ol type="a">
<li><strong>Operational (Required):</strong> Account management, order processing, product catalog access, user authentication</li>
<li><strong>Analytics (Optional):</strong> Usage pattern analysis, performance optimization, understanding user behavior</li>
<li><strong>Marketing (Optional):</strong> Promotional communications, personalized offers, newsletters</li>
<li><strong>Payment (Required):</strong> Payment processing via Midtrans (third party)</li>
</ol>
</section>

<section class="legal-basis">
<h2>3. LEGAL BASIS</h2>
<p>We process your personal data based on:</p>
<ul>
<li>Your explicit consent (Article 20 UU PDP) - collected during registration and checkout</li>
<li>Performance of service contract</li>
<li>Legal obligations (tax data retention, audit records)</li>
</ul>
</section>

<section class="retention">
<h2>4. RETENTION PERIODS</h2>
<ul>
<li><strong>Active Accounts:</strong> Data retained while account is active</li>
<li><strong>Deleted Accounts:</strong> 90-day grace period before permanent deletion</li>
<li><strong>Guest Orders:</strong> 5 years (Indonesian tax requirements)</li>
<li><strong>Audit Logs:</strong> 7 years (Indonesian compliance standards)</li>
<li><strong>Temporary Data:</strong> 48 hours (verification tokens, password reset tokens)</li>
</ul>
</section>

<section class="third-party">
<h2>5. THIRD-PARTY DATA SHARING</h2>
<p>We share your data with:</p>
<div class="third-party-box">
<h4>Midtrans (Payment Processor)</h4>
<p>Processes payment transactions</p>
<ul>
<li><strong>Privacy Policy:</strong> <a href="https://midtrans.com/privacy-notice" target="_blank" rel="noopener noreferrer">https://midtrans.com/privacy-notice</a></li>
<li><strong>Data Location:</strong> Indonesia</li>
<li><strong>Legal Basis:</strong> Explicit consent for payment processing</li>
</ul>
</div>
<p class="highlight"><strong>We do NOT sell your personal data to third parties.</strong></p>
</section>

<section class="security">
<h2>6. SECURITY MEASURES</h2>
<p>We implement the following security measures:</p>
<ul>
<li><strong>Encryption at Rest:</strong> All personal data is encrypted in storage</li>
<li><strong>Access Controls:</strong> Multi-tenant isolation, role-based access control (RBAC)</li>
<li><strong>Audit Logging:</strong> Immutable audit trail for all data access</li>
<li><strong>Log Masking:</strong> Sensitive data is masked in application logs</li>
</ul>
</section>

<section class="your-rights">
<h2>7. YOUR RIGHTS (Articles 3-6 UU PDP)</h2>
<p>You have the right to:</p>
<ol type="a">
<li><strong>Access:</strong> View all personal data we hold about you</li>
<li><strong>Correction:</strong> Update inaccurate or incomplete information</li>
<li><strong>Deletion:</strong> Request deletion of your personal data (with exceptions)</li>
<li><strong>Consent Withdrawal:</strong> Withdraw optional consent at any time</li>
<li><strong>Data Portability:</strong> Export your data in JSON format</li>
<li><strong>Complaint:</strong> File a complaint with data protection authorities</li>
</ol>
<p>To exercise your rights, contact: <a href="mailto:posku-service@proton.me">posku-service@proton.me</a></p>
</section>

<section class="deletion-limits">
<h2>8. DELETION LIMITATIONS</h2>
<p>We cannot delete data if:</p>
<ul>
<li>Required for legal obligations (tax records, audit records)</li>
<li>Required for legal claims or dispute resolution</li>
<li>Required for security and fraud prevention</li>
</ul>
</section>

<section class="complaint-process">
<h2>9. COMPLAINT PROCESS</h2>
<p>If you have a complaint about our data processing:</p>
<ol>
<li>Contact us at <a href="mailto:posku-service@proton.me">posku-service@proton.me</a></li>
<li>We will respond within 14 calendar days (Article 6 UU PDP)</li>
<li>If not satisfied, file a complaint with the Ministry of Communication and Informatics</li>
</ol>
</section>

<section class="policy-changes">
<h2>10. POLICY CHANGES</h2>
<p>We may update this policy from time to time:</p>
<ul>
<li><strong>Minor Updates (v1.x.x):</strong> Non-blocking notification</li>
<li><strong>Major Updates (v2.0.0):</strong> Re-consent required</li>
</ul>
</section>

<section class="contact">
<h2>11. CONTACT INFORMATION</h2>
<ul>
<li><strong>Email:</strong> <a href="mailto:posku-service@proton.me">posku-service@proton.me</a></li>
<li><strong>Response Time:</strong> 14 calendar days</li>
<li><strong>Address:</strong> - </li>
</ul>
</section>

<section class="acknowledgment">
<p class="disclaimer">By using our services, you acknowledge that you have read and understood this Privacy Policy.</p>
</section>
</div>',
    
    '2026-01-02 00:00:00+00',
    TRUE,  -- is_major_update (initial version)
    TRUE   -- is_current
)
ON CONFLICT (version) DO NOTHING;