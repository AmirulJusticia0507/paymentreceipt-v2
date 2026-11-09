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
   'Pengunjung', 'viewer'),
  ('requester', 'requester@asi-pudjiastuti-aviation.co.id',
   '$argon2id$v=19$m=19456,t=2,p=1$jtN84i07H0n2b2NPHdBmMA$cCoZsTMtwt+uBrFbjACy15ElO9NPL824eFY/wHf+Hjw',
   'Staf Pemohon (Requester)', 'requester');

-- ---------------------------------------------------------------------------
-- Master data
-- ---------------------------------------------------------------------------
INSERT INTO department (kode, nama, keterangan) VALUES
  ('GA',    'General Affairs',                'Bagian umum, fasilitas kantor, rumah tangga & aset pangkalan'),
  ('OPS',   'Flight & Ground Operations',     'Operasional penerbangan, dispatch, pangkalan & ground handling'),
  ('ENG',   'Maintenance & Engineering',      'Perawatan armada pesawat, teknisi, avionik & hanggar MRO'),
  ('FIN',   'Finance & Accounting',           'Keuangan, kasir pangkalan, pembukuan & akuntansi'),
  ('HRD',   'Human Resources & Training',     'SDM, personalia, rekrutmen, pelatihan & expatriate'),
  ('IT',    'Information Technology',         'Teknologi informasi, infrastruktur jaringan & sistem aplikasi'),
  ('SQS',   'Safety, Quality & Security',     'Keselamatan penerbangan, quality assurance & keamanan bandara'),
  ('COM',   'Commercial, Cargo & Ticketing',  'Komersial, penjualan tiket, carter pesawat & kargo perintis'),
  ('PROC',  'Procurement & Supply Chain',     'Pengadaan barang, suku cadang & manajemen rantai pasok');

UPDATE users SET department_id = (SELECT id FROM department WHERE kode = 'GA')
WHERE username = 'requester';

INSERT INTO satuan (kode, nama) VALUES
  ('PCS', 'Pcs'), ('BOX', 'Box'), ('LUS', 'Lusin'), ('RIM', 'Rim'),
  ('SET', 'Set'), ('KG',  'Kilogram'), ('LTR', 'Liter'), ('PAK', 'Paket'), ('GAL', 'Galon');

INSERT INTO categories (kode, nama, keterangan) VALUES
  ('MRO',  'Suku Cadang & Maintenance Pesawat',   'Aircraft spare parts, consumable hanggar, pelumas & tools aviasi'),
  ('GSE',  'Ground Support Equipment',            'Peralatan penunjang darat, chock, towbar, rambu apron & bagasi'),
  ('ATK',  'Alat Tulis Kantor & Dokumen Flight',  'Stationery kantor, flight log book, formulir manifes & tiket'),
  ('ELEK', 'Elektronik, Radio & IT Hardware',     'Komputer, radio VHF aviasi, handy talkie & jaringan base'),
  ('APAR', 'Keselamatan, K3 & Fire Protection',   'Alat pemadam api, rompi hi-vis, safety shoes & perlengkapan K3'),
  ('FURN', 'Furnitur & Inventaris Pangkalan',     'Meja kerja, kursi ruang tunggu penumpang & lemari station'),
  ('SERA', 'Seragam & Perlengkapan Personel',     'Seragam pilot, cabin crew, teknisi hanggar & ID card staf'),
  ('KOM',  'Telekomunikasi & Layanan Satelit',    'Internet pangkalan, telepon satelit, pulsa radio & komunikasi'),
  ('MB',   'Konsumsi & Pantry Pangkalan',         'Snack penerbangan perintis, air galon & pantry kantor/base'),
  ('JASA', 'Jasa Pihak Ketiga & Perawatan',       'Servis fasilitas, ekspedisi kargo, kalibrasi alat & perbaikan gedung'),
  ('LAIN', 'Kebutuhan Operasional Lain-lain',     'Pengeluaran taktis dan operasional pangkalan yang tidak terkategori');

INSERT INTO base (kode, nama, keterangan) VALUES
  ('PNG', 'Base Pangandaran',   'Headquarters & Main Engineering Base - Jawa Barat'),
  ('HLP', 'Base Jakarta',        'Halim Perdanakusuma (HLP) - DKI Jakarta'),
  ('KNO', 'Base Medan',          'Kualanamu / Medan - Sumatera Utara'),
  ('PDG', 'Base Padang',         'Minangkabau - Sumatera Barat'),
  ('PKU', 'Base Pekanbaru',      'Sultan Syarif Kasim II - Riau'),
  ('SIQ', 'Base Dabo Singkep',   'Dabo Singkep - Kepulauan Riau'),
  ('TRK', 'Base Tarakan',        'Juwata - Kalimantan Utara'),
  ('LNU', 'Base Malinau',        'Kolonel Robert Atty Bessing - Kalimantan Utara'),
  ('AAP', 'Base Samarinda',      'APT Pranoto - Kalimantan Timur'),
  ('BDJ', 'Base Banjarmasin',    'Syamsudin Noor - Kalimantan Selatan'),
  ('BPN', 'Base Balikpapan',     'Sultan Aji Muhammad Sulaiman Sepinggan - Kalimantan Timur'),
  ('MXB', 'Base Masamba',        'Andi Djemma - Sulawesi Selatan'),
  ('MDC', 'Base Manado',         'Sam Ratulangi - Sulawesi Utara'),
  ('KOE', 'Base Kupang',         'El Tari - Nusa Tenggara Timur'),
  ('TTE', 'Base Ternate',        'Sultan Babullah - Maluku Utara'),
  ('AMQ', 'Base Ambon',          'Pattimura - Maluku'),
  ('SOQ', 'Base Sorong',         'Domine Eduard Osok - Papua Barat Daya'),
  ('MKW', 'Base Manokwari',      'Rendani - Papua Barat'),
  ('BIK', 'Base Biak',           'Frans Kaisiepo - Papua'),
  ('NBX', 'Base Nabire',         'Douw Aturure - Papua Tengah'),
  ('TIM', 'Base Timika',         'Mozes Kilangin - Papua Tengah'),
  ('DJJ', 'Base Sentani',        'Sentani / Jayapura - Papua'),
  ('WMX', 'Base Wamena',         'Wamena - Papua Pegunungan'),
  ('DEX', 'Base Dekai',          'Nop Goliat Dekai - Papua Pegunungan'),
  ('MKQ', 'Base Merauke',        'Mopah - Papua Selatan');

INSERT INTO requests (kode, nama, department_id, keterangan)
SELECT 'REQ-GA-001', 'Pengadaan ATK GA', d.id, 'ATK untuk kebutuhan GA'
FROM department d WHERE d.kode = 'GA';

INSERT INTO requests (kode, nama, department_id, keterangan)
SELECT 'REQ-OPS-001', 'Perlengkapan Operasional', d.id, 'Kebutuhan lapangan'
FROM department d WHERE d.kode = 'OPS';

INSERT INTO request_base (kode, nama, jabatan, base_id, keterangan)
SELECT 'RB-PNG-01', 'Operasional Kantor Pusat', 'GA Officer', b.id, 'Kebutuhan umum Head Office Pangandaran'
FROM base b WHERE b.kode = 'PNG';

INSERT INTO request_base (kode, nama, jabatan, base_id, keterangan)
SELECT 'RB-HLP-01', 'Ground Handling & Operasional', 'Station Manager', b.id, 'Kebutuhan operasional Base Halim'
FROM base b WHERE b.kode = 'HLP';

INSERT INTO request_base (kode, nama, jabatan, base_id, keterangan)
SELECT 'RB-TIM-01', 'Flight Ops & Maintenance', 'Base Coordinator', b.id, 'Kebutuhan operasional pangkalan Timika'
FROM base b WHERE b.kode = 'TIM';

INSERT INTO request_base (kode, nama, jabatan, base_id, keterangan)
SELECT 'RB-DJJ-01', 'Perintis & Kargo Papua', 'Operations Officer', b.id, 'Kebutuhan pangkalan Sentani Jayapura'
FROM base b WHERE b.kode = 'DJJ';

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
