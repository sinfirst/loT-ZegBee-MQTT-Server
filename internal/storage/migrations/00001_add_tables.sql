-- +goose Up
-- Создание таблиц

CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    telegram_id INT NOT NULL,
    username TEXT NOT NULL,
    hub_id TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE devices (
    device_id TEXT PRIMARY KEY NOT NULL,
    ieee_addr TEXT UNIQUE NOT NULL,
    user_id INT,
    hub_id TEXT NOT NULL,
    model_id TEXT NOT NULL,
    device_type INT NOT NULL,
    device_status INT NOT NULL,
    device_online BOOLEAN NOT NULL,
    battery_percentage INT NOT NULL, 
    battery_last_seen_timestamp TIMESTAMP WITH TIME ZONE,
    last_seen INT NOT NULL,
    last_seen_timestamp TIMESTAMP WITH TIME ZONE,
    link_quality INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    hub_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    link_quality INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS devices;