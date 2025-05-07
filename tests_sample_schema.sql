-- comment
CREATE TABLE authors (
  id   BIGSERIAL PRIMARY KEY,
  name text      NOT NULL,
  bio  text
);

DROP TABLE IF EXISTS authors;

CREATE TABLE "another_table" (
  id   BIGSERIAL PRIMARY KEY,
  name text      NOT NULL,
  "bio"  text
);

CREATE TABLE "with_schema"."another_table" (
  id   BIGSERIAL PRIMARY KEY,
  name text      NOT NULL,
  unknownType some unknown types and keywords,
  "bio"  text
);