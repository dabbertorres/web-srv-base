create database if not exists web;
use web;

create table if not exists users
(
    name     varchar(32) primary key,
    password binary(60) not null,
    admin    bool       not null,
    enabled  bool       not null
);

create table if not exists visits
(
    user      varchar(32)   null,
    time      datetime      not null,
    ip        varbinary(16) not null,
    userAgent varchar(64),
    path      varchar(32)   not null,
    action    enum ('GET', 'HEAD', 'POST', 'PUT', 'DELETE', 'CONNECT', 'OPTIONS', 'TRACE', 'PATCH'),
    params    json,
    foreign key (user) references users (name)
        on delete set null
        on update cascade
);
