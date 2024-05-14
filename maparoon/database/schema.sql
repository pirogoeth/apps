pragma foreign_keys = on;

create table if not exists networks (
    id integer primary key,
    name string not null,
    address string not null,
    cidr integer not null,
    comments text,
    attributes text
);
create index if not exists idx_address on networks(address);
create index if not exists idx_address_cidr on networks(address, cidr);

create table if not exists hosts (
    id integer primary key,
    network_id integer,
    address blob not null collate binary,
    comments text,
    attributes text,

    foreign key (network_id) references networks(id) on delete cascade
);
create index if not exists idx_address on hosts(address);

create table if not exists host_ports (
    address blob not null collate binary,
    port integer not null,
    protocol text not null,
    comments text,
    attributes text,

    primary key (address, port, protocol),
    foreign key (address) references hosts(address) on delete cascade
);
create index if not exists idx_address on host_ports(address);
create index if not exists idx_address_port on host_ports(address, port);
create index if not exists idx_address_port_proto on host_ports(address, port, protocol);