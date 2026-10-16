-- 0009_sources.up.sql
-- Sumber / toko pembelian.
--
-- Legacy: `sourcesproc` dengan kolom `source` VARCHAR bebas. Nilainya bisa apa saja -
-- legacy tidak punya CHECK constraint, jadi data rusak bisa masuk diam-diam.
-- Index unik (source, namatoko) yang di-comment di kode legacy juga tidak pernah dibuat.
--
-- Di sini: enum + CHECK constraint per mode. Lihat function_resources.php legacy:
-- mode online  -> simpan namabarang + linkpembelian
-- mode offline -> simpan namatoko + nomortelptoko + rekeningtoko + namabarang

CREATE TYPE source_type AS ENUM ('online', 'offline');

CREATE TABLE sources (
    id               BIGSERIAL PRIMARY KEY,
    source           source_type  NOT NULL,
    nama_barang      VARCHAR(200) NOT NULL,
    nama_toko        VARCHAR(150),
    no_telp          VARCHAR(30),
    no_rekening      VARCHAR(50),
    link_pembelian   TEXT,
    keterangan       TEXT         NOT NULL DEFAULT '',
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Mode ONLINE wajib punya link pembelian
    CONSTRAINT sources_online_requires_link CHECK (
        source <> 'online' OR link_pembelian IS NOT NULL
    ),
    -- Mode OFFLINE wajib punya nama toko
    CONSTRAINT sources_offline_requires_shop CHECK (
        source <> 'offline' OR nama_toko IS NOT NULL
    ),
    -- Kolom yang tidak relevan dengan mode harus kosong (bukan diisi asal)
    CONSTRAINT sources_offline_no_link  CHECK (source <> 'offline' OR link_pembelian IS NULL),
    CONSTRAINT sources_online_no_shop   CHECK (source <> 'online'  OR nama_toko IS NULL),
    -- Format no. telp: angka, spasi, +, -, tanda kurung (maks 30)
    CONSTRAINT sources_phone_format CHECK (
        no_telp IS NULL OR no_telp ~ '^[0-9+()\- ]{6,30}$'
    )
);

-- Partial unique index: satu toko offline tidak boleh dobel dengan nama + rekening sama.
-- COALESCE dipakai karena kolom nullable - unik tetap berlaku walau NULL.
CREATE UNIQUE INDEX idx_sources_offline_shop ON sources (COALESCE(nama_toko, ''), COALESCE(no_rekening, ''))
    WHERE source = 'offline';
CREATE UNIQUE INDEX idx_sources_online_link ON sources (link_pembelian)
    WHERE source = 'online' AND link_pembelian IS NOT NULL;

CREATE INDEX idx_sources_active ON sources (is_active);
CREATE INDEX idx_sources_barang_trgm ON sources USING GIN (nama_barang gin_trgm_ops);