-- +goose Up
-- Создание таблиц

-- Создаем тип для Acceleration
CREATE TYPE acceleration_type AS (
    x FLOAT,
    y FLOAT,
    z FLOAT
);

-- Создаем тип для Angle
CREATE TYPE angle_type AS (
    pitch FLOAT,
    roll FLOAT
);

-- Создаем тип для Battery
CREATE TYPE battery_type AS (
    voltage FLOAT,
    percentage INT
);

CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    telegram_id INT NOT NULL,
    username TEXT NOT NULL,
    hub_id TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE devices (
    device_id TEXT PRIMARY KEY,
    user_id INT NOT NULL,
    hub_id TEXT NOT NULL,
    device_type TEXT NOT NULL,
    last_event TEXT,
    battery battery_type,
    signal_strength INT,
    orientation_state TEXT NOT NULL,
    sensor_status TEXT NOT NULL,
    last_seen TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    hub_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_confidence DECIMAL(3,2) NOT NULL,
    signal_strength INT,
    temperature DECIMAL(4,1),
    acceleration acceleration_type,
    angle angle_type,
    battery battery_type,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(device_id) ON DELETE CASCADE
);

CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    notification_type VARCHAR(100) NOT NULL,
    message TEXT,
    sent_status VARCHAR(50) NOT NULL DEFAULT 'sent',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(device_id) ON DELETE CASCADE
);


-- Индексы


-- +goose Down
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS devices;