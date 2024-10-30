pragma foreign_keys = on;

create table if not exists projects (
    id integer primary key,
    name text not null,
    description text not null,
    created_at integer not null default 0,
    updated_at integer not null default 0
);

create trigger if not exists trg_project_created
after insert on projects
for each row
begin
    update projects
    set
        created_at = strftime('%s', 'now'),
        updated_at = strftime('%s', 'now')
    where id = new.id and created_at = 0;
end;

create trigger if not exists trg_project_updated
after update on projects
for each row
when (
    old.created_at = new.created_at
    and old.updated_at = new.updated_at
)
begin
    update projects
    set
        created_at = strftime('%s', 'now'),
        updated_at = strftime('%s', 'now')
    where id = new.id;
end;