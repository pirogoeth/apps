-- +goose Up

create virtual table vec_memories using vec0(
  user_id integer not null partition key,
  memory_id integer not null,
  memory_embedding float[768],
);

-- +goose Down

drop table vec_memories;
