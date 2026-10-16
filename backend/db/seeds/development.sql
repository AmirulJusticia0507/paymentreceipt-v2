-- development.sql — data awal untuk lingkungan lokal.
--
-- JANGAN pernah dijalankan di production. Password di sini lemah dan disengaja
-- supaya mudah diuji; seluruh kredensial lain harus datang dari environment variable.
--
-- Password di-hash dengan argon2id (OWASP recommended params: m=19 MiB, t=2, p=1).
-- Password yang dipakai:
--   admin      -> admin123
--   ga_staff   -> ga_staff123
--   approver   -> approver123

BEGIN;

-- ---------------------------------------------------------------------------
-- User
-- ---------------------------------------------------------------------------
INSERT INTO users (username, email, password_hash, full_name, role) VALUES
  ('admin',    'admin@asi-pudjiastuti-aviation.co.id',
   '$argon2id$v=19$m=19456,t=2,p=1$l8pm3voJKir4Erh65IQpuA$FBlDnyPZn2NAo2ee3m38SfX421wzBbdlLv00ZmQauVU',
   'Administrator', 'admin'),
  ('ga_staff', 'ga@asi-pudjiastuti-aviation.co.id',
   '$argon2id$v=19$m=19456,t=2,p=1$wh+vd29aLLSJMFnxIUSe0A$UJSk6TjNhSKKa80mz6XpJj5dZSiYBe+AwdwncUszMBU',
   'Staf General Affairs', 'ga_staff'),
  ('approver', 'approver@asi-pudjiastuti-aviation.co.id',
   '$argon2id$v=19$m=19456,t=2,p=1$jtN84i07H0n2b2NPHdBmMA$cCoZsTMtwt+uBrFbjACy15ElO9NPL824eFY/wHf+Hjw',
   'Manajer Approver', 'approver'),
  ('viewer',   'viewer@asi-pudjiastuti-aviation.co.id',
   '$argon2id$v=19$m=19456,t=2,p=1$jtN84i07H0n2b2NPHdBmMA$cCoZsTMtwt+uBrFbjACy15ElO9NPL824eFY/wHf+Hjw',
   'Pengunjung', 'viewer');

-- ---------------------------------------------------------------------------
-- Master data
-- ---------------------------------------------------------------------------
INSERT INTO department (kode, nama, keterangan) VALUES
  ('GA',    'General Affairs', 'Bagian umum'),
  ('OPS',   'Operations',      'Operasional'),
  ('ENG',   'Engineering',     'Teknik'),
  ('FIN',   'Finance',         'Keuangan'),
  ('HRD',   'Human Resources', 'Sumber daya manusia'),
  ('SEC',   'Security',        'Keamanan');

INSERT INTO satuan (kode, nama) VALUES
  ('PCS', 'Pcs'), ('BOX', 'Box'), ('LUS', 'Lusin'), ('RIM', 'Rim'),
  ('SET', 'Set'), ('KG',  'Kilogram'), ('LTR', 'Liter'), ('PAK', 'Paket'), ('GAL', 'Galon');

INSERT INTO categories (kode, nama, keterangan) VALUES
  ('ATK', 'Alat Tulis Kantor',    'ATK & Stationery'),
  ('ELEK', 'Elektronik',          'Perangkat elektronik'),
  ('FURN', 'Furnitur',            'Perabot kantor'),
  ('MB',  'Perlengkapan Makan',  'Kebutuhan makan & minum'),
  ('APAR', 'Perlengkapan/APAR',   'Alat pemadam & keselamatan'),
  ('KOM',  'Komunikasi',          'Telepon, internet, pulse'),
  ('JASA', 'Jasa',                'Jasa pihak ketiga'),
  ('LAIN', 'Lain-lain',           'Tidak terkategori');

INSERT INTO base (kode, nama, keterangan) VALUES
  ('PK', 'Perlengkapan Kantor',  'Barang habis pakai untuk operasional'),
  ('PJ', 'Peralatan & Jewelry',  'Peralatan emitir dan aksesori Reciprocal'),
  ('SR', 'Servis & Perawatan',   'Jasa servis dan perawatan aset'),
  ('OT', 'Operasional Toolkit',  'Kebutuhan khusus operasional');

INSERT INTO requests (kode, nama, department_id, keterangan)
SELECT 'REQ-GA-001', 'Pengadaan ATK GA', d.id, 'ATK untuk kebutuhan GA'
FROM department d WHERE d.kode = 'GA';

INSERT INTO requests (kode, nama, department_id, keterangan)
SELECT 'REQ-OPS-001', 'Perlengkapan Operasional', d.id, 'Kebutuhan lapangan'
FROM department d WHERE d.kode = 'OPS';

INSERT INTO request_base (kode, nama, jabatan, base_id, keterangan)
SELECT 'RB-001', 'Pengadaan Komputer', 'GA Officer', b.id, 'Unit kerja untuk request komputer'
FROM base b WHERE b.kode = 'PK';

INSERT INTO request_base (kode, nama, jabatan, base_id, keterangan)
SELECT 'RB-002', 'Servis AC Ruangan', 'Maintenance Lead', b.id, 'Request perawatan AC'
FROM base b WHERE b.kode = 'SR';

-- Mode online -> wajib link, nama_toko harus NULL
INSERT INTO sources (source, nama_barang, nama_toko, link_pembelian, keterangan) VALUES
  ('online', 'Kertas A4 80gr 5 rim', NULL,
   'https://www.tokopedia.com/example/a4-80gr', 'Pembelian via marketplace'),
  ('online', 'Printer Laser Toner', NULL,
   'https://www.shopee.co.id/example/toner', 'Pembelian via marketplace'),
  ('online', 'Kabel LAN Cat6 100m', NULL,
   'https://bukalapak.com/example/kabel-cat6', 'Pembelian via marketplace');

-- Mode offline -> wajib nama_toko, link harus NULL
INSERT INTO sources (source, nama_barang, nama_toko, no_telp, no_rekening, keterangan) VALUES
  ('offline', 'Kursi Kerja Ergonomis', 'CV Sinar furnishing',
   '021-1234567', '1234567890', 'Vendor furnishing lokal'),
  ('offline', 'Air Mineral Galon 19L', 'CV Sumber Air Jaya',
   '021-7654321', '0987654321', 'Vendor air minum'),
  ('offline', 'Servis AC Cleaning', 'PT Sejuk-AC',
   '021-5551234', '1122334455', 'Vendor servis AC');

-- ---------------------------------------------------------------------------
-- Contoh transaksi
--
-- Menutup setiap status dengan satu baris supaya filter & dashboard bisa
-- langsung diuji tanpa harus klik-klik.
-- ---------------------------------------------------------------------------

-- pending: 2 items
INSERT INTO payment_receipt
  (kode, request_id, department_id, source_id, category_id, unit_id,
   tanggal_pembelian, usage, deskripsi_items, jumlah_items, hargaitems, price_estimate, keterangan, status)
SELECT 'PR-2026-000001', r.id, d.id, s.id, c.id, u.id,
       CURRENT_DATE - 3, 'GA Office', 'Kertas A4 80gr', 10, 55000, 50000,
       'Pembelian rutin ATK', 'pending'
FROM requests r, department d, sources s, categories c, satuan u
WHERE r.kode = 'REQ-GA-001' AND d.kode = 'GA'
  AND s.nama_barang = 'Kertas A4 80gr 5 rim' AND c.kode = 'ATK' AND u.kode = 'RIM';

INSERT INTO payment_receipt
  (kode, request_id, department_id, source_id, category_id, unit_id,
   tanggal_pembelian, usage, deskripsi_items, jumlah_items, hargaitems, price_estimate, keterangan, status)
SELECT 'PR-2026-000002', r.id, d.id, s.id, c.id, u.id,
       CURRENT_DATE - 2, 'GA Office', 'Pulpen dan pencil', 5, 35000, 30000,
       'ATK tulis', 'pending'
FROM requests r, department d, sources s, categories c, satuan u
WHERE r.kode = 'REQ-GA-001' AND d.kode = 'GA'
  AND s.nama_barang = 'Printer Laser Toner' AND c.kode = 'ATK' AND u.kode = 'BOX';

-- checked -> processed -> finished: riwayat lengkap
INSERT INTO payment_receipt
  (kode, request_id, department_id, source_id, category_id, unit_id,
   tanggal_pembelian, usage, deskripsi_items, jumlah_items, hargaitems, price_estimate, keterangan, status)
SELECT 'PR-2026-000003', r.id, d.id, s.id, c.id, u.id,
       CURRENT_DATE - 10, 'Ruang Raport', 'Kursi kerja ergonomis', 12, 1850000, 1750000,
       'Pengadaan furnitur ruang raport', 'finished'
FROM requests r, department d, sources s, categories c, satuan u
WHERE r.kode = 'REQ-GA-001' AND d.kode = 'GA'
  AND s.nama_barang = 'Kursi Kerja Ergonomis' AND c.kode = 'FURN' AND u.kode = 'PCS';

-- cancelled
INSERT INTO payment_receipt
  (kode, request_id, department_id, source_id, category_id, unit_id,
   tanggal_pembelian, usage, deskripsi_items, jumlah_items, hargaitems, price_estimate, keterangan, status)
SELECT 'PR-2026-000004', r.id, d.id, s.id, c.id, u.id,
       CURRENT_DATE - 5, 'GA Office', 'Air galon 19L', 3, 185000, 175000,
       'Dibatalkan: contoh status terminal', 'cancelled'
FROM requests r, department d, sources s, categories c, satuan u
WHERE r.kode = 'REQ-GA-001' AND d.kode = 'GA'
  AND s.nama_barang = 'Air Mineral Galon 19L' AND c.kode = 'MB' AND u.kode = 'GAL';

COMMIT;

-- ---------------------------------------------------------------------------
-- Status history untuk transaksi finished (PR-2026-000003).
--
-- Disisipkan terpisah karena butuh user_id aktual.
-- ---------------------------------------------------------------------------
INSERT INTO status_history (payment_receipt_id, from_status, to_status, changed_by, note)
SELECT p.id, NULL, 'pending', u.id, 'Dibuat oleh GA staff'
FROM payment_receipt p, users u WHERE p.kode = 'PR-2026-000003' AND u.username = 'ga_staff';

INSERT INTO status_history (payment_receipt_id, from_status, to_status, changed_by, note)
SELECT p.id, 'pending', 'checked', u.id, 'Verifikasi GA'
FROM payment_receipt p, users u WHERE p.kode = 'PR-2026-000003' AND u.username = 'approver';

INSERT INTO status_history (payment_receipt_id, from_status, to_status, changed_by, note)
SELECT p.id, 'checked', 'processed', u.id, 'Purchase order dikirim'
FROM payment_receipt p, users u WHERE p.kode = 'PR-2026-000003' AND u.username = 'ga_staff';

INSERT INTO status_history (payment_receipt_id, from_status, to_status, changed_by, note)
SELECT p.id, 'processed', 'finished', u.id, 'Barang diterima lengkap'
FROM payment_receipt p, users u WHERE p.kode = 'PR-2026-000003' AND u.username = 'approver';

-- Riwayat untuk pending (masih satu entry: saat dibuat)
INSERT INTO status_history (payment_receipt_id, from_status, to_status, changed_by, note)
SELECT p.id, NULL, 'pending', u.id, 'Dibuat oleh GA staff'
FROM payment_receipt p, users u WHERE p.kode = 'PR-2026-000001' AND u.username = 'ga_staff';

INSERT INTO status_history (payment_receipt_id, from_status, to_status, changed_by, note)
SELECT p.id, NULL, 'pending', u.id, 'Dibuat oleh GA staff'
FROM payment_receipt p, users u WHERE p.kode = 'PR-2026-000002' AND u.username = 'ga_staff';

INSERT INTO status_history (payment_receipt_id, from_status, to_status, changed_by, note)
SELECT p.id, NULL, 'cancelled', u.id, 'Dibatalkan oleh approver'
FROM payment_receipt p, users u WHERE p.kode = 'PR-2026-000004' AND u.username = 'approver';

-- ---------------------------------------------------------------------------
-- Arsip PR
-- ---------------------------------------------------------------------------
INSERT INTO pr_archive
  (kode, payment_receipt_id, tanggal_selesai, nama_request, manager_approved, approved_by,
   description, quantity, harga_rencana, harga_aktual)
SELECT 'PRA-2026-000001', p.id, NOW() - INTERVAL '2 days',
       'Pengadaan Furnitur Ruang Raport', 'Bpk. Kepala GA', u.id,
       'Pengadaan 12 kursi kerja ergonomis untuk ruang raport', 12, 21000000, p.total_harga
FROM payment_receipt p, users u
WHERE p.kode = 'PR-2026-000003' AND u.username = 'approver';
