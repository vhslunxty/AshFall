package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const (
	energyRegenEvery = 5 * time.Minute // 1 point d'énergie toutes les 5 minutes
	exploreCost      = 1
	fightCost        = 2
)

// GameError = refus "normal" du jeu (pas assez d'énergie...), pas une panne serveur.
type GameError struct{ Msg string }

func (e *GameError) Error() string { return e.Msg }

func gameErr(format string, a ...any) error {
	return &GameError{Msg: fmt.Sprintf(format, a...)}
}

func xpNeeded(level int) int { return level * 50 }

// regen recalcule l'énergie récupérée depuis la dernière action (calcul "paresseux").
func regen(p *Player) {
	now := time.Now().UTC()
	if p.Energy >= p.MaxEnergy {
		p.Energy = p.MaxEnergy
		p.lastTick = now
		return
	}
	ticks := int(now.Sub(p.lastTick) / energyRegenEvery)
	if ticks <= 0 {
		return
	}
	p.Energy += ticks
	if p.Energy >= p.MaxEnergy {
		p.Energy = p.MaxEnergy
		p.lastTick = now
	} else {
		p.lastTick = p.lastTick.Add(time.Duration(ticks) * energyRegenEvery)
	}
}

func spend(p *Player, cost int) error {
	if p.Energy < cost {
		return gameErr("⚡ Pas assez d'énergie (%d requis). Tu en récupères 1 point toutes les 5 minutes.", cost)
	}
	p.Energy -= cost
	return nil
}

func gainXP(p *Player, xp int) string {
	p.XP += xp
	var sb strings.Builder
	for p.XP >= xpNeeded(p.Level) {
		p.XP -= xpNeeded(p.Level)
		p.Level++
		p.MaxHP += 10
		p.HP = p.MaxHP
		sb.WriteString(fmt.Sprintf("\n🎉 **Niveau %d !** PV max : %d, tu es soigné.", p.Level, p.MaxHP))
		if p.Level%3 == 0 {
			p.MaxEnergy++
			sb.WriteString(" ⚡ +1 énergie max.")
		}
	}
	return sb.String()
}

func doExplore(tx *sql.Tx, p *Player) (string, error) {
	if err := spend(p, exploreCost); err != nil {
		return "", err
	}

	roll := rand.Intn(100)
	switch {
	case roll < 35: // matériaux
		it, err := randomItem(tx, "material", 1000)
		if err != nil {
			return "", err
		}
		qty := 1 + rand.Intn(3)
		if err := addItem(tx, p.DiscordID, it.ID, qty); err != nil {
			return "", err
		}
		return fmt.Sprintf("🗺️ Tu fouilles une carcasse de voiture et récupères %s **%s** x%d.", it.Emoji, it.Name, qty), nil

	case roll < 50: // nourriture
		it, err := randomItem(tx, "food", 1000)
		if err != nil {
			return "", err
		}
		qty := 1 + rand.Intn(2)
		if err := addItem(tx, p.DiscordID, it.ID, qty); err != nil {
			return "", err
		}
		return fmt.Sprintf("🏚️ Dans une cabane abandonnée, tu trouves %s **%s** x%d.", it.Emoji, it.Name, qty), nil

	case roll < 62: // or
		gold := (5 + rand.Intn(16)) * p.Level
		p.Gold += gold
		return fmt.Sprintf("💰 Un cadavre serrait une bourse : **+%d or**.", gold), nil

	case roll < 67: // arme rare
		it, err := randomItem(tx, "weapon", p.Level*4+2)
		if err != nil {
			return "", err
		}
		if err := addItem(tx, p.DiscordID, it.ID, 1); err != nil {
			return "", err
		}
		return fmt.Sprintf("✨ Trouvaille rare ! Tu déterres %s **%s** (⚔️ +%d).", it.Emoji, it.Name, it.Power), nil

	case roll < 82: // piège
		dmg := 5 + rand.Intn(11)
		if dmg >= p.HP {
			dmg = p.HP - 1
		}
		p.HP -= dmg
		return fmt.Sprintf("🪤 Un piège à loups se referme sur ta jambe : **-%d PV**.", dmg), nil

	default:
		flavors := []string{
			"🌫️ Tu ne croises que du brouillard et des corbeaux. Rien à signaler.",
			"🌲 La forêt est silencieuse… trop silencieuse. Tu rentres bredouille.",
			"🏜️ Des heures de marche pour rien, mais tu as appris le terrain.",
		}
		return flavors[rand.Intn(len(flavors))], nil
	}
}

func doFight(tx *sql.Tx, p *Player) (string, error) {
	if p.HP < 10 {
		return "", gameErr("🩸 Tu es trop blessé pour te battre (%d PV). Mange pour te soigner !", p.HP)
	}
	if err := spend(p, fightCost); err != nil {
		return "", err
	}

	enemies := []string{"🧟 Zombie", "🐺 Loup affamé", "🥷 Pillard", "👹 Mutant", "🦇 Chauve-souris géante"}
	enemy := enemies[rand.Intn(len(enemies))]

	weapon := bestWeaponPower(tx, p.DiscordID)
	enemyHP := 20 + p.Level*8
	enemyDmg := 3 + p.Level*2
	playerDmg := 5 + p.Level*2 + weapon

	startHP := p.HP
	rounds := 0
	for p.HP > 0 && enemyHP > 0 {
		rounds++
		enemyHP -= max(1, playerDmg+rand.Intn(5)-2)
		if enemyHP <= 0 {
			break
		}
		p.HP -= max(1, enemyDmg+rand.Intn(3)-1)
	}
	if p.HP < 0 {
		p.HP = 0
	}

	// Défaite
	if p.HP <= 0 {
		lost := p.Gold / 4
		p.Gold -= lost
		p.HP = max(1, p.MaxHP/2)
		return fmt.Sprintf("☠️ **%s** t'a terrassé en %d rounds… Tu te réveilles plus tard, blessé (%d PV)%s.",
			enemy, rounds, p.HP, goldLost(lost)), nil
	}

	// Victoire
	xp := 10 + p.Level*5 + rand.Intn(6)
	gold := (3 + rand.Intn(8)) * p.Level
	p.Gold += gold
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⚔️ Tu bats **%s** en %d rounds (-%d PV). **+%d XP**, **+%d or**.",
		enemy, rounds, startHP-p.HP, xp, gold))

	if rand.Intn(100) < 40 {
		typ := "material"
		if rand.Intn(2) == 0 {
			typ = "food"
		}
		if it, err := randomItem(tx, typ, 1000); err == nil {
			if err := addItem(tx, p.DiscordID, it.ID, 1); err != nil {
				return "", err
			}
			sb.WriteString(fmt.Sprintf("\n🎁 Butin : %s **%s**.", it.Emoji, it.Name))
		}
	}
	sb.WriteString(gainXP(p, xp))
	return sb.String(), nil
}

func goldLost(lost int) string {
	if lost <= 0 {
		return ""
	}
	return fmt.Sprintf(" et tu as perdu %d or", lost)
}

func doEat(tx *sql.Tx, p *Player) (string, error) {
	if p.HP >= p.MaxHP {
		return "", gameErr("😋 Tu es déjà en pleine forme.")
	}

	var it Item
	err := tx.QueryRow(
		`SELECT i.id, i.name, i.emoji, i.type, i.power
		 FROM inventory inv JOIN items i ON i.id = inv.item_id
		 WHERE inv.discord_id = ? AND i.type = 'food' AND inv.quantity > 0
		 ORDER BY i.power ASC LIMIT 1`, p.DiscordID,
	).Scan(&it.ID, &it.Name, &it.Emoji, &it.Type, &it.Power)
	if errors.Is(err, sql.ErrNoRows) {
		return "", gameErr("🎒 Tu n'as aucune nourriture. Explore pour en trouver !")
	}
	if err != nil {
		return "", err
	}

	heal := min(it.Power, p.MaxHP-p.HP)
	p.HP += heal
	if err := removeOne(tx, p.DiscordID, it.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s Tu manges **%s** et récupères **%d PV**.", it.Emoji, it.Name, heal), nil
}
