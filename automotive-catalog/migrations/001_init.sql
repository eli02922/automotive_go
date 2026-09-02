-- ============================================================
-- Automotive Catalog - Initial Schema
-- ============================================================

BEGIN;

-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
-- Categories (hierarchical)
-- ============================================================
CREATE TABLE categories (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    name        TEXT NOT NULL,
    parent_id   TEXT REFERENCES categories(id),
    level       INT  NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_categories_parent_id ON categories(parent_id);

-- ============================================================
-- Products
-- ============================================================
CREATE TABLE products (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    part_number  TEXT NOT NULL,
    brand        TEXT NOT NULL,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    category_id  TEXT NOT NULL REFERENCES categories(id),
    status       TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive','discontinued')),
    weight       NUMERIC(10,3),
    dimensions   JSONB NOT NULL DEFAULT '{}',
    attributes   JSONB NOT NULL DEFAULT '{}',
    images       TEXT[] NOT NULL DEFAULT '{}',
    msrp         NUMERIC(12,2),
    cost         NUMERIC(12,2),
    deleted_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_products_part_number UNIQUE (part_number)
);

-- Indexes for high-volume query patterns
CREATE INDEX idx_products_brand          ON products(brand) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_category_id    ON products(category_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_status         ON products(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_created_at     ON products(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_fts            ON products USING GIN (to_tsvector('english', name || ' ' || description));
CREATE INDEX idx_products_attributes_gin ON products USING GIN (attributes);

-- Staging table for bulk upserts (used by BulkUpsert in repo)
CREATE TABLE products_staging (LIKE products INCLUDING ALL);

-- ============================================================
-- Vehicles
-- ============================================================
CREATE TABLE vehicles (
    id        TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    year      SMALLINT NOT NULL,
    make      TEXT     NOT NULL,
    model     TEXT     NOT NULL,
    sub_model TEXT     NOT NULL DEFAULT '',
    engine    TEXT     NOT NULL DEFAULT '',
    region    TEXT     NOT NULL DEFAULT 'NA',
    CONSTRAINT uq_vehicles UNIQUE (year, make, model, sub_model, engine)
);

CREATE INDEX idx_vehicles_ymm ON vehicles(year, make, model);

-- ============================================================
-- Fitments (ACES standard product-to-vehicle applications)
-- ============================================================
CREATE TABLE fitments (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    year       SMALLINT NOT NULL,
    make       TEXT     NOT NULL,
    model      TEXT     NOT NULL,
    sub_model  TEXT     NOT NULL DEFAULT '',
    engine     TEXT     NOT NULL DEFAULT '',
    notes      TEXT     NOT NULL DEFAULT '',
    qualifiers JSONB    NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_fitments UNIQUE (product_id, year, make, model, sub_model, engine)
);

CREATE INDEX idx_fitments_product_id ON fitments(product_id);
CREATE INDEX idx_fitments_ymm        ON fitments(year, make, model);

-- ============================================================
-- Inventory
-- ============================================================
CREATE TABLE inventory (
    product_id          TEXT    NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    warehouse_code      TEXT    NOT NULL,
    quantity_on_hand    INT     NOT NULL DEFAULT 0,
    quantity_reserved   INT     NOT NULL DEFAULT 0,
    quantity_available  INT     GENERATED ALWAYS AS (quantity_on_hand - quantity_reserved) STORED,
    reorder_point       INT     NOT NULL DEFAULT 0,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (product_id, warehouse_code)
);

CREATE INDEX idx_inventory_product_id ON inventory(product_id);

-- ============================================================
-- Outbox (transactional outbox pattern for Kafka reliability)
-- ============================================================
CREATE TABLE outbox_events (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    event_type   TEXT        NOT NULL,
    aggregate_id TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    published    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_outbox_unpublished ON outbox_events(created_at) WHERE published = FALSE;

COMMIT;
