SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS players (
  discord_id       BIGINT UNSIGNED PRIMARY KEY,
  hp               INT NOT NULL DEFAULT 100,
  max_hp           INT NOT NULL DEFAULT 100,
  energy           INT NOT NULL DEFAULT 10,
  max_energy       INT NOT NULL DEFAULT 10,
  xp               INT NOT NULL DEFAULT 0,
  level            INT NOT NULL DEFAULT 1,
  gold             INT NOT NULL DEFAULT 0,
  last_energy_tick DATETIME NOT NULL,
  created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS items (
  id     INT AUTO_INCREMENT PRIMARY KEY,
  name   VARCHAR(50) NOT NULL UNIQUE,
  emoji  VARCHAR(16) NOT NULL DEFAULT '📦',
  type   ENUM('weapon','food','material') NOT NULL,
  power  INT NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS inventory (
  discord_id BIGINT UNSIGNED NOT NULL,
  item_id    INT NOT NULL,
  quantity   INT NOT NULL DEFAULT 1,
  PRIMARY KEY (discord_id, item_id),
  FOREIGN KEY (discord_id) REFERENCES players(discord_id) ON DELETE CASCADE,
  FOREIGN KEY (item_id) REFERENCES items(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Armes (power = dégâts bonus)
INSERT INTO items (name, emoji, type, power) VALUES
  ('Couteau rouillé', '🔪', 'weapon', 2),
  ('Batte cloutée',   '🏏', 'weapon', 5),
  ('Machette',        '🗡️', 'weapon', 9),
  ('Arbalète',        '🏹', 'weapon', 14),
  ('Fusil de chasse', '🔫', 'weapon', 22);

-- Nourriture (power = PV rendus)
INSERT INTO items (name, emoji, type, power) VALUES
  ('Pomme',              '🍎', 'food', 8),
  ('Ration',             '🥫', 'food', 15),
  ('Viande grillée',     '🍖', 'food', 30),
  ('Trousse de soins',   '🩹', 'food', 50);

-- Matériaux (pour le craft futur)
INSERT INTO items (name, emoji, type, power) VALUES
  ('Bois',      '🪵', 'material', 0),
  ('Ferraille', '🔩', 'material', 0),
  ('Tissu',     '🧵', 'material', 0),
  ('Pierre',    '🪨', 'material', 0);
