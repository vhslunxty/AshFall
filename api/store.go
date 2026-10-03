package main

import (
	"database/sql"
	"errors"
	"time"
)

// querier est satisfait par *sql.DB et *sql.Tx.
type querier interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
}

type Item struct {
	ID    int
	Name  string
	Emoji string
	Type  string
	Power int
}

type InvItem struct {
	ItemID   int    `json:"item_id"`
	Name     string `json:"name"`
	Emoji    string `json:"emoji"`
	Type     string `json:"type"`
	Power    int    `json:"power"`
	Quantity int    `json:"quantity"`
}

type Player struct {
	DiscordID     uint64    `json:"discord_id,string"`
	HP            int       `json:"hp"`
	MaxHP         int       `json:"max_hp"`
	Energy        int       `json:"energy"`
	MaxEnergy     int       `json:"max_energy"`
	XP            int       `json:"xp"`
	Level         int       `json:"level"`
	Gold          int       `json:"gold"`
	NextLevelXP   int       `json:"next_level_xp"`
	NextEnergySec int       `json:"next_energy_sec"`
	Inventory     []InvItem `json:"inventory"`

	lastTick time.Time
}

var errNoPlayer = errors.New("player not found")

func loadPlayer(q querier, id uint64, forUpdate bool) (*Player, error) {
	query := `SELECT hp, max_hp, energy, max_energy, xp, level, gold, last_energy_tick
	          FROM players WHERE discord_id = ?`
	if forUpdate {
		query += " FOR UPDATE"
	}
	p := &Player{DiscordID: id}
	err := q.QueryRow(query, id).Scan(
		&p.HP, &p.MaxHP, &p.Energy, &p.MaxEnergy, &p.XP, &p.Level, &p.Gold, &p.lastTick,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNoPlayer
	}
	return p, err
}

func savePlayer(q querier, p *Player) error {
	_, err := q.Exec(
		`UPDATE players SET hp=?, max_hp=?, energy=?, max_energy=?, xp=?, level=?, gold=?, last_energy_tick=?
		 WHERE discord_id=?`,
		p.HP, p.MaxHP, p.Energy, p.MaxEnergy, p.XP, p.Level, p.Gold, p.lastTick, p.DiscordID,
	)
	return err
}

func loadInventory(q querier, id uint64) ([]InvItem, error) {
	rows, err := q.Query(
		`SELECT i.id, i.name, i.emoji, i.type, i.power, inv.quantity
		 FROM inventory inv JOIN items i ON i.id = inv.item_id
		 WHERE inv.discord_id = ? AND inv.quantity > 0
		 ORDER BY FIELD(i.type, 'weapon', 'food', 'material'), i.power DESC, i.name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []InvItem{}
	for rows.Next() {
		var it InvItem
		if err := rows.Scan(&it.ItemID, &it.Name, &it.Emoji, &it.Type, &it.Power, &it.Quantity); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func addItem(q querier, id uint64, itemID, qty int) error {
	_, err := q.Exec(
		`INSERT INTO inventory (discord_id, item_id, quantity) VALUES (?, ?, ?) AS n
		 ON DUPLICATE KEY UPDATE quantity = inventory.quantity + n.quantity`,
		id, itemID, qty)
	return err
}

func removeOne(q querier, id uint64, itemID int) error {
	_, err := q.Exec(
		`UPDATE inventory SET quantity = quantity - 1
		 WHERE discord_id = ? AND item_id = ? AND quantity > 0`, id, itemID)
	return err
}

func randomItem(q querier, typ string, maxPower int) (*Item, error) {
	it := &Item{}
	err := q.QueryRow(
		`SELECT id, name, emoji, type, power FROM items
		 WHERE type = ? AND power <= ? ORDER BY RAND() LIMIT 1`, typ, maxPower,
	).Scan(&it.ID, &it.Name, &it.Emoji, &it.Type, &it.Power)
	return it, err
}

func itemByName(q querier, name string) (*Item, error) {
	it := &Item{}
	err := q.QueryRow(
		`SELECT id, name, emoji, type, power FROM items WHERE name = ?`, name,
	).Scan(&it.ID, &it.Name, &it.Emoji, &it.Type, &it.Power)
	return it, err
}

func bestWeaponPower(q querier, id uint64) int {
	var power int
	_ = q.QueryRow(
		`SELECT COALESCE(MAX(i.power), 0)
		 FROM inventory inv JOIN items i ON i.id = inv.item_id
		 WHERE inv.discord_id = ? AND i.type = 'weapon' AND inv.quantity > 0`, id,
	).Scan(&power)
	return power
}

func giveStarter(q querier, id uint64) error {
	starter := []struct {
		name string
		qty  int
	}{
		{"Couteau rouillé", 1},
		{"Ration", 2},
	}
	for _, s := range starter {
		it, err := itemByName(q, s.name)
		if err != nil {
			return err
		}
		if err := addItem(q, id, it.ID, s.qty); err != nil {
			return err
		}
	}
	return nil
}
