-- 0004_satuan.up.sql
-- Satuan barang: pcs, box, rim, lusin, dll.
--
-- CATATAN: di legacy, tabel `satuan` di-JOIN di banyak query (data_paymentreceipt.php,
-- printdocuments.php, updatepaymentreceipt.php) tapi TIDAK punya form insert maupun
-- handler CRUD. Artinya user tidak bisa menambah satuan baru sama sekali.
-- Bug ini yang diperbaiki di sini - lihat T4.4/T4.5.

CREATE TABLE satuan (
    id          BIGSERIAL PRIMARY KEY,
    kode        VARCHAR(20)  NOT NULL,
    nama        VARCHAR(50)  NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT satuan_kode_key UNIQUE (kode),
    CONSTRAINT satuan_kode_format CHECK (kode ~ '^[A-Z0-9]{1,20}$')
);

CREATE INDEX idx_satuan_active_nama ON satuan (nama) WHERE is_active;