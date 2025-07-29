-- +goose Up

pragma foreign_keys = on;

create table if not exists users (
  id integer primary key,
  name text unique not null,
  created_at datetime default (strftime('%s', 'now')),
  created_date text as (date(created_at, 'auto'))
);

-- +goose StatementBegin
create trigger if not exists trg_users_source_creation
after insert on users
for each row
begin
  insert into sources (id, user_id, description, protected) values (
    'user_memory',
    new.id,
    format('Permanent memories for user %s (id %d)', new.name, new.id),
    true
  );
end;
-- +goose StatementEnd

create table if not exists sources (
  id text not null,
  user_id integer not null,
  description text,
  protected boolean default false,

  primary key (user_id, id),
  foreign key (user_id) references users(id) on delete cascade on update cascade 
);

-- +goose StatementBegin
create trigger if not exists trg_source_protected_deletion
before delete on sources
for each row
when old.protected=true
begin
  select raise(rollback, 'protected: Memory source is protected, unset `protected` to delete');
end;
-- +goose StatementEnd

create table if not exists memories (
  id integer primary key,
  user_id integer not null,
  source_id text not null,
  created_at datetime default (strftime('%s', 'now')),
  created_date text as (date(created_at, 'auto')),
  memory text not null,

  foreign key (user_id) references users(id) on delete cascade on update cascade,
  foreign key (source_id, user_id) references sources(id, user_id) on delete cascade on update cascade
);

create virtual table if not exists memories_fts using fts5(
  user_id,
  source_id,
  memory,
  content=memories,
  content_rowid=id
);

-- +goose StatementBegin
create trigger if not exists trg_memories_ai_source
before insert on memories
for each row
begin
  insert or ignore into sources (id, user_id) values (new.source_id, new.user_id);
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger if not exists trg_memories_ai_fts
after insert on memories
for each row
begin
  insert into memories_fts (rowid, user_id, source_id, memory)
    values (new.id, new.user_id, new.source_id, new.memory);
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger if not exists trg_memories_au_fts
after update on memories
for each row
begin
  insert into memories_fts (memories_fts, rowid, user_id, source_id, memory) values (
    'delete',
    old.id, old.user_id, old.source_id, old.memory
  );
  insert into memories_fts (rowid, user_id, source_id, memory) values (
    new.id, new.user_id, new.source_id, new.memory
  );
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger if not exists trg_memories_ad_fts
after delete on memories
for each row
begin
  insert into memories_fts (memories_fts, rowid, user_id, source_id, memory) values (
    'delete',
    old.id, old.user_id, old.source_id, old.memory
  );
end;
-- +goose StatementEnd
