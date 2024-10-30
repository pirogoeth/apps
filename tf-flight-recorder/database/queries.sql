-- name: CreateProject :one
insert into projects (
    name, description, created_at, updated_at
) values (
    ?, ?, ?, ?
)
returning *;

-- name: GetProjectById :one
select * from projects
where id = ? limit 1;

-- name: GetProjectByName :one
select * from projects
where name = ? limit 1;

-- name: GetProjects :many
select * from projects;

