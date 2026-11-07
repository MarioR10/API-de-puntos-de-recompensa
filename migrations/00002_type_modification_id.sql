-- +goose Up
ALTER TABLE users ALTER COLUMN id DROP DEFAULT;
DROP SEQUENCE  users_id_seq;
ALTER TABLE users ALTER COLUMN id TYPE UUID USING gen_random_uuid();
-- +goose Down
CREATE SEQUENCE users_id_seq;
ALTER TABLE users ALTER COLUMN id TYPE bigint USING nextval('users_id_seq');
ALTER TABLE users ALTER COLUMN id SET DEFAULT nextval('users_id_seq');
ALTER SEQUENCE users_id_seq OWNED BY users.id;