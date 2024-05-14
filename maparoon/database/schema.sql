pragma foreign_keys = on;

create table if not exists networks (
    id integer primary key,
    name text not null,
    address text not null,
    cidr integer not null,
    comments text not null,
    attributes text not null
);
create index if not exists idx_networks_address on networks(address);
create index if not exists idx_networks_address_cidr on networks(address, cidr);

create table if not exists hosts (
    id integer primary key,
    network_id integer not null,
    address text unique not null,
    comments text,
    attributes text,

    foreign key (network_id) references networks(id) on delete cascade on update cascade
);
create index if not exists idx_hosts_address on hosts(address);

create table if not exists host_ports (
    address text not null,
    port integer not null,
    protocol text not null,
    comments text,
    attributes text,

    primary key (address, port, protocol),
    foreign key (address) references hosts(address) on delete cascade on update cascade
);
create index if not exists idx_host_ports_address on host_ports(address);
create index if not exists idx_host_ports_address_port on host_ports(address, port);
create index if not exists idx_host_ports_address_port_proto on host_ports(address, port, protocol);