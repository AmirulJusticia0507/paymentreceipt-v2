--
-- PostgreSQL database dump
--

\restrict h0jcDBPnkyszZE6MRcKpgNuWIFarm64y8Ve2ASPVkxngMUUHwJA71i5Kw5jPcZq

-- Dumped from database version 18.6
-- Dumped by pg_dump version 18.6

-- Name: citext; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;


--
-- Name: EXTENSION citext; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION citext IS 'data type for case-insensitive character strings';


--
-- Name: pg_trgm; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;


--
-- Name: EXTENSION pg_trgm; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: payment_status; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.payment_status AS ENUM (
    'pending',
    'checked',
    'processed',
    'finished',
    'cancelled'
);


--
-- Name: source_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.source_type AS ENUM (
    'online',
    'offline'
);


--
-- Name: user_role; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.user_role AS ENUM (
    'admin',
    'ga_staff',
    'approver',
    'viewer'
);


--
-- Name: set_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;


-- Name: base; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.base (
    id bigint NOT NULL,
    kode character varying(30) NOT NULL,
    nama character varying(150) NOT NULL,
    keterangan text DEFAULT ''::text NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT base_kode_format CHECK (((kode)::text ~ '^[A-Z0-9-]{2,30}$'::text))
);


--
-- Name: base_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: base_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: categories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.categories (
    id bigint NOT NULL,
    kode character varying(30) NOT NULL,
    nama character varying(100) NOT NULL,
    keterangan character varying(255) DEFAULT ''::character varying NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT categories_kode_format CHECK (((kode)::text ~ '^[A-Z0-9-]{2,30}$'::text))
);


--
-- Name: categories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: categories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: department; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.department (
    id bigint NOT NULL,
    kode character varying(30) NOT NULL,
    nama character varying(100) NOT NULL,
    keterangan character varying(255) DEFAULT ''::character varying NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT department_kode_format CHECK (((kode)::text ~ '^[A-Z0-9-]{2,30}$'::text))
);


--
-- Name: department_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: department_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: payment_receipt; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.payment_receipt (
    id bigint NOT NULL,
    kode character varying(30) NOT NULL,
    request_id bigint NOT NULL,
    department_id bigint NOT NULL,
    source_id bigint,
    category_id bigint,
    unit_id bigint,
    tanggal_pembelian date NOT NULL,
    usage character varying(150) DEFAULT ''::character varying NOT NULL,
    deskripsi_items character varying(255) NOT NULL,
    jumlah_items integer NOT NULL,
    hargaitems numeric(15,2) NOT NULL,
    total_harga numeric(15,2) GENERATED ALWAYS AS (((jumlah_items)::numeric * hargaitems)) STORED,
    price_estimate numeric(15,2),
    keterangan text DEFAULT ''::text NOT NULL,
    status public.payment_status DEFAULT 'pending'::public.payment_status NOT NULL,
    inputdate timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT payment_receipt_date_not_future CHECK ((tanggal_pembelian <= CURRENT_DATE)),
    CONSTRAINT payment_receipt_deskripsi_len CHECK (((char_length((deskripsi_items)::text) >= 1) AND (char_length((deskripsi_items)::text) <= 255))),
    CONSTRAINT payment_receipt_estimate_positive CHECK (((price_estimate IS NULL) OR (price_estimate >= (0)::numeric))),
    CONSTRAINT payment_receipt_price_positive CHECK ((hargaitems > (0)::numeric)),
    CONSTRAINT payment_receipt_qty_positive CHECK ((jumlah_items > 0)),
    CONSTRAINT payment_receipt_usage_len CHECK ((char_length((usage)::text) <= 150))
);


--
-- Name: payment_receipt_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: payment_receipt_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: pr_archive; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pr_archive (
    id bigint NOT NULL,
    kode character varying(30) NOT NULL,
    payment_receipt_id bigint NOT NULL,
    tanggal_selesai timestamp with time zone DEFAULT now() NOT NULL,
    nama_request character varying(150) NOT NULL,
    manager_approved character varying(150) DEFAULT ''::character varying NOT NULL,
    approved_by bigint,
    description text DEFAULT ''::text NOT NULL,
    quantity integer DEFAULT 0 NOT NULL,
    harga_rencana numeric(15,2) DEFAULT 0 NOT NULL,
    harga_aktual numeric(15,2) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT pr_archive_aktual_nonneg CHECK ((harga_aktual >= (0)::numeric)),
    CONSTRAINT pr_archive_quantity_positive CHECK ((quantity >= 0)),
    CONSTRAINT pr_archive_rencana_nonneg CHECK ((harga_rencana >= (0)::numeric))
);


--
-- Name: pr_archive_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: pr_archive_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: refresh_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.refresh_tokens (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    user_agent text,
    ip_address inet,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT refresh_tokens_expires_future CHECK ((expires_at > created_at)),
    CONSTRAINT refresh_tokens_revoked_not_future CHECK (((revoked_at IS NULL) OR (revoked_at <= now())))
);


--
-- Name: refresh_tokens_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: refresh_tokens_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: request_base; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.request_base (
    id bigint NOT NULL,
    kode character varying(30) NOT NULL,
    nama character varying(150) NOT NULL,
    jabatan character varying(100) DEFAULT ''::character varying NOT NULL,
    base_id bigint NOT NULL,
    keterangan text DEFAULT ''::text NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT request_base_kode_format CHECK (((kode)::text ~ '^[A-Z0-9-]{2,30}$'::text))
);


--
-- Name: request_base_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: request_base_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.requests (
    id bigint NOT NULL,
    kode character varying(30) NOT NULL,
    nama character varying(150) NOT NULL,
    department_id bigint NOT NULL,
    keterangan text DEFAULT ''::text NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT requests_kode_format CHECK (((kode)::text ~ '^[A-Z0-9-]{2,30}$'::text))
);


--
-- Name: requests_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: satuan; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.satuan (
    id bigint NOT NULL,
    kode character varying(20) NOT NULL,
    nama character varying(50) NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT satuan_kode_format CHECK (((kode)::text ~ '^[A-Z0-9]{1,20}$'::text))
);


--
-- Name: satuan_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: satuan_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: seq_payment_receipt_kode; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: seq_pr_archive_kode; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sources (
    id bigint NOT NULL,
    source public.source_type NOT NULL,
    nama_barang character varying(200) NOT NULL,
    nama_toko character varying(150),
    no_telp character varying(30),
    no_rekening character varying(50),
    link_pembelian text,
    keterangan text DEFAULT ''::text NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT sources_offline_no_link CHECK (((source <> 'offline'::public.source_type) OR (link_pembelian IS NULL))),
    CONSTRAINT sources_offline_requires_shop CHECK (((source <> 'offline'::public.source_type) OR (nama_toko IS NOT NULL))),
    CONSTRAINT sources_online_no_shop CHECK (((source <> 'online'::public.source_type) OR (nama_toko IS NULL))),
    CONSTRAINT sources_online_requires_link CHECK (((source <> 'online'::public.source_type) OR (link_pembelian IS NOT NULL))),
    CONSTRAINT sources_phone_format CHECK (((no_telp IS NULL) OR ((no_telp)::text ~ '^[0-9+()\- ]{6,30}$'::text)))
);


--
-- Name: sources_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: sources_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: status_history; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.status_history (
    id bigint NOT NULL,
    payment_receipt_id bigint NOT NULL,
    from_status public.payment_status,
    to_status public.payment_status NOT NULL,
    changed_by bigint NOT NULL,
    note text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT status_history_changed CHECK (((from_status IS NULL) OR (from_status <> to_status)))
);


--
-- Name: status_history_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: status_history_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    username public.citext NOT NULL,
    email public.citext NOT NULL,
    password_hash text NOT NULL,
    full_name character varying(150) DEFAULT ''::character varying NOT NULL,
    role public.user_role DEFAULT 'viewer'::public.user_role NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    last_login_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT users_email_format CHECK ((email OPERATOR(public.~) '^[^@\s]+@[^@\s]+\.[^@\s]+$'::public.citext)),
    CONSTRAINT users_fullname_len CHECK ((char_length((full_name)::text) <= 150)),
    CONSTRAINT users_username_format CHECK ((username OPERATOR(public.~) '^[a-zA-Z0-9._-]{3,50}$'::public.citext))
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

-- Name: base id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.base ALTER COLUMN id SET DEFAULT nextval('public.base_id_seq'::regclass);


--
-- Name: categories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories ALTER COLUMN id SET DEFAULT nextval('public.categories_id_seq'::regclass);


--
-- Name: department id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.department ALTER COLUMN id SET DEFAULT nextval('public.department_id_seq'::regclass);


--
-- Name: payment_receipt id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_receipt ALTER COLUMN id SET DEFAULT nextval('public.payment_receipt_id_seq'::regclass);


--
-- Name: pr_archive id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pr_archive ALTER COLUMN id SET DEFAULT nextval('public.pr_archive_id_seq'::regclass);


--
-- Name: refresh_tokens id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens ALTER COLUMN id SET DEFAULT nextval('public.refresh_tokens_id_seq'::regclass);


--
-- Name: request_base id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.request_base ALTER COLUMN id SET DEFAULT nextval('public.request_base_id_seq'::regclass);


--
-- Name: requests id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.requests ALTER COLUMN id SET DEFAULT nextval('public.requests_id_seq'::regclass);


--
-- Name: satuan id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.satuan ALTER COLUMN id SET DEFAULT nextval('public.satuan_id_seq'::regclass);


--
-- Name: sources id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sources ALTER COLUMN id SET DEFAULT nextval('public.sources_id_seq'::regclass);


--
-- Name: status_history id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.status_history ALTER COLUMN id SET DEFAULT nextval('public.status_history_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: base base_kode_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.base
    ADD CONSTRAINT base_kode_key UNIQUE (kode);


--
-- Name: base base_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.base
    ADD CONSTRAINT base_pkey PRIMARY KEY (id);


--
-- Name: categories categories_kode_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_kode_key UNIQUE (kode);


--
-- Name: categories categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_pkey PRIMARY KEY (id);


--
-- Name: department department_kode_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.department
    ADD CONSTRAINT department_kode_key UNIQUE (kode);


--
-- Name: department department_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.department
    ADD CONSTRAINT department_pkey PRIMARY KEY (id);


--
-- Name: payment_receipt payment_receipt_kode_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_receipt
    ADD CONSTRAINT payment_receipt_kode_key UNIQUE (kode);


--
-- Name: payment_receipt payment_receipt_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_receipt
    ADD CONSTRAINT payment_receipt_pkey PRIMARY KEY (id);


--
-- Name: pr_archive pr_archive_kode_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pr_archive
    ADD CONSTRAINT pr_archive_kode_key UNIQUE (kode);


--
-- Name: pr_archive pr_archive_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pr_archive
    ADD CONSTRAINT pr_archive_pkey PRIMARY KEY (id);


--
-- Name: refresh_tokens refresh_tokens_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_hash_key UNIQUE (token_hash);


--
-- Name: refresh_tokens refresh_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);


--
-- Name: request_base request_base_kode_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.request_base
    ADD CONSTRAINT request_base_kode_key UNIQUE (kode);


--
-- Name: request_base request_base_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.request_base
    ADD CONSTRAINT request_base_pkey PRIMARY KEY (id);


--
-- Name: requests requests_kode_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.requests
    ADD CONSTRAINT requests_kode_key UNIQUE (kode);


--
-- Name: requests requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.requests
    ADD CONSTRAINT requests_pkey PRIMARY KEY (id);


--
-- Name: satuan satuan_kode_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.satuan
    ADD CONSTRAINT satuan_kode_key UNIQUE (kode);


--
-- Name: satuan satuan_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.satuan
    ADD CONSTRAINT satuan_pkey PRIMARY KEY (id);


--
-- Name: sources sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sources
    ADD CONSTRAINT sources_pkey PRIMARY KEY (id);


--
-- Name: status_history status_history_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.status_history
    ADD CONSTRAINT status_history_pkey PRIMARY KEY (id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: idx_base_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_base_active ON public.base USING btree (is_active);


--
-- Name: idx_base_nama_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_base_nama_trgm ON public.base USING gin (nama public.gin_trgm_ops);


--
-- Name: idx_categories_active_nama; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_categories_active_nama ON public.categories USING btree (nama) WHERE is_active;


--
-- Name: idx_department_active_nama; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_department_active_nama ON public.department USING btree (nama) WHERE is_active;


--
-- Name: idx_payment_receipt_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_receipt_category ON public.payment_receipt USING btree (category_id);


--
-- Name: idx_payment_receipt_department; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_receipt_department ON public.payment_receipt USING btree (department_id);


--
-- Name: idx_payment_receipt_dept_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_receipt_dept_status ON public.payment_receipt USING btree (department_id, status);


--
-- Name: idx_payment_receipt_deskripsi_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_receipt_deskripsi_trgm ON public.payment_receipt USING gin (deskripsi_items public.gin_trgm_ops);


--
-- Name: idx_payment_receipt_request; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_receipt_request ON public.payment_receipt USING btree (request_id);


--
-- Name: idx_payment_receipt_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_receipt_source ON public.payment_receipt USING btree (source_id);


--
-- Name: idx_payment_receipt_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_receipt_status ON public.payment_receipt USING btree (status);


--
-- Name: idx_payment_receipt_tanggal; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_receipt_tanggal ON public.payment_receipt USING btree (tanggal_pembelian DESC, id DESC);


--
-- Name: idx_payment_receipt_unit; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_receipt_unit ON public.payment_receipt USING btree (unit_id);


--
-- Name: idx_pr_archive_approver; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pr_archive_approver ON public.pr_archive USING btree (approved_by);


--
-- Name: idx_pr_archive_manager_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pr_archive_manager_trgm ON public.pr_archive USING gin (manager_approved public.gin_trgm_ops);


--
-- Name: idx_pr_archive_receipt; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pr_archive_receipt ON public.pr_archive USING btree (payment_receipt_id);


--
-- Name: idx_pr_archive_tanggal; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pr_archive_tanggal ON public.pr_archive USING btree (tanggal_selesai DESC, id DESC);


--
-- Name: idx_refresh_tokens_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_active ON public.refresh_tokens USING btree (user_id, expires_at) WHERE (revoked_at IS NULL);


--
-- Name: idx_refresh_tokens_expires; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_expires ON public.refresh_tokens USING btree (expires_at);


--
-- Name: idx_refresh_tokens_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_user ON public.refresh_tokens USING btree (user_id);


--
-- Name: idx_request_base_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_request_base_active ON public.request_base USING btree (is_active);


--
-- Name: idx_request_base_base; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_request_base_base ON public.request_base USING btree (base_id);


--
-- Name: idx_request_base_nama_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_request_base_nama_trgm ON public.request_base USING gin (nama public.gin_trgm_ops);


--
-- Name: idx_requests_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_requests_active ON public.requests USING btree (is_active);


--
-- Name: idx_requests_department; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_requests_department ON public.requests USING btree (department_id);


--
-- Name: idx_requests_nama_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_requests_nama_trgm ON public.requests USING gin (nama public.gin_trgm_ops);


--
-- Name: idx_satuan_active_nama; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_satuan_active_nama ON public.satuan USING btree (nama) WHERE is_active;


--
-- Name: idx_sources_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sources_active ON public.sources USING btree (is_active);


--
-- Name: idx_sources_barang_trgm; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sources_barang_trgm ON public.sources USING gin (nama_barang public.gin_trgm_ops);


--
-- Name: idx_sources_offline_shop; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_sources_offline_shop ON public.sources USING btree (COALESCE(nama_toko, ''::character varying), COALESCE(no_rekening, ''::character varying)) WHERE (source = 'offline'::public.source_type);


--
-- Name: idx_sources_online_link; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_sources_online_link ON public.sources USING btree (link_pembelian) WHERE ((source = 'online'::public.source_type) AND (link_pembelian IS NOT NULL));


--
-- Name: idx_status_history_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_status_history_created ON public.status_history USING btree (created_at);


--
-- Name: idx_status_history_receipt; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_status_history_receipt ON public.status_history USING btree (payment_receipt_id, created_at DESC);


--
-- Name: idx_status_history_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_status_history_user ON public.status_history USING btree (changed_by);


--
-- Name: idx_users_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_active ON public.users USING btree (is_active);


--
-- Name: idx_users_last_login; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_last_login ON public.users USING btree (last_login_at DESC NULLS LAST);


--
-- Name: idx_users_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_role ON public.users USING btree (role) WHERE is_active;


--
-- Name: base trg_base_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

-- Name: categories trg_categories_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

-- Name: department trg_department_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

-- Name: payment_receipt trg_payment_receipt_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

-- Name: pr_archive trg_pr_archive_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

-- Name: request_base trg_request_base_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

-- Name: requests trg_requests_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

-- Name: satuan trg_satuan_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

-- Name: sources trg_sources_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

-- Name: users trg_users_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

-- Name: payment_receipt payment_receipt_category_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_receipt
    ADD CONSTRAINT payment_receipt_category_fk FOREIGN KEY (category_id) REFERENCES public.categories(id) ON DELETE RESTRICT;


--
-- Name: payment_receipt payment_receipt_department_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_receipt
    ADD CONSTRAINT payment_receipt_department_fk FOREIGN KEY (department_id) REFERENCES public.department(id) ON DELETE RESTRICT;


--
-- Name: payment_receipt payment_receipt_request_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_receipt
    ADD CONSTRAINT payment_receipt_request_fk FOREIGN KEY (request_id) REFERENCES public.requests(id) ON DELETE RESTRICT;


--
-- Name: payment_receipt payment_receipt_source_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_receipt
    ADD CONSTRAINT payment_receipt_source_fk FOREIGN KEY (source_id) REFERENCES public.sources(id) ON DELETE RESTRICT;


--
-- Name: payment_receipt payment_receipt_unit_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_receipt
    ADD CONSTRAINT payment_receipt_unit_fk FOREIGN KEY (unit_id) REFERENCES public.satuan(id) ON DELETE RESTRICT;


--
-- Name: pr_archive pr_archive_approved_by_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pr_archive
    ADD CONSTRAINT pr_archive_approved_by_fk FOREIGN KEY (approved_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: pr_archive pr_archive_receipt_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pr_archive
    ADD CONSTRAINT pr_archive_receipt_fk FOREIGN KEY (payment_receipt_id) REFERENCES public.payment_receipt(id) ON DELETE RESTRICT;


--
-- Name: refresh_tokens refresh_tokens_user_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: request_base request_base_base_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.request_base
    ADD CONSTRAINT request_base_base_fk FOREIGN KEY (base_id) REFERENCES public.base(id) ON DELETE RESTRICT;


--
-- Name: requests requests_department_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.requests
    ADD CONSTRAINT requests_department_fk FOREIGN KEY (department_id) REFERENCES public.department(id) ON DELETE RESTRICT;


--
-- Name: status_history status_history_receipt_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.status_history
    ADD CONSTRAINT status_history_receipt_fk FOREIGN KEY (payment_receipt_id) REFERENCES public.payment_receipt(id) ON DELETE CASCADE;


--
-- Name: status_history status_history_user_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.status_history
    ADD CONSTRAINT status_history_user_fk FOREIGN KEY (changed_by) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- PostgreSQL database dump complete
--

\unrestrict h0jcDBPnkyszZE6MRcKpgNuWIFarm64y8Ve2ASPVkxngMUUHwJA71i5Kw5jPcZq

