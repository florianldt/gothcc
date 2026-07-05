-- +goose Up
create table if not exists messages (
    id         bigint generated always as identity primary key,
    body       text not null,
    created_at timestamptz not null default now()
);

-- +goose Down
drop table if exists messages;
