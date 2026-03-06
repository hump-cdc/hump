CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email      text        NOT NULL UNIQUE,
    name       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version    bigint      NOT NULL DEFAULT 1
);

CREATE TABLE posts (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id  bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      text        NOT NULL,
    body       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version    bigint      NOT NULL DEFAULT 1
);

CREATE INDEX posts_author_id_idx ON posts (author_id);

CREATE TABLE comments (
    id         bigint      GENERATED ALWAYS AS IDENTITY,
    post_id    uuid        NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id  bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version    bigint      NOT NULL DEFAULT 1,
    PRIMARY KEY (id, author_id)
) PARTITION BY HASH (author_id);

CREATE TABLE comments_p00 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER  0);
CREATE TABLE comments_p01 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER  1);
CREATE TABLE comments_p02 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER  2);
CREATE TABLE comments_p03 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER  3);
CREATE TABLE comments_p04 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER  4);
CREATE TABLE comments_p05 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER  5);
CREATE TABLE comments_p06 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER  6);
CREATE TABLE comments_p07 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER  7);
CREATE TABLE comments_p08 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER  8);
CREATE TABLE comments_p09 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER  9);
CREATE TABLE comments_p10 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER 10);
CREATE TABLE comments_p11 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER 11);
CREATE TABLE comments_p12 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER 12);
CREATE TABLE comments_p13 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER 13);
CREATE TABLE comments_p14 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER 14);
CREATE TABLE comments_p15 PARTITION OF comments FOR VALUES WITH (MODULUS 16, REMAINDER 15);

CREATE INDEX comments_post_id_idx   ON comments (post_id);
CREATE INDEX comments_author_id_idx ON comments (author_id);

CREATE TABLE events (
    id         bigint      GENERATED ALWAYS AS IDENTITY,
    time       timestamptz NOT NULL DEFAULT now(),
    user_id    bigint,
    kind       text        NOT NULL,
    payload    jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    version    bigint      NOT NULL DEFAULT 1,
    PRIMARY KEY (id, time)
);

SELECT create_hypertable('events', 'time');

CREATE INDEX events_kind_time_idx    ON events (kind, time DESC);
CREATE INDEX events_user_id_time_idx ON events (user_id, time DESC);
