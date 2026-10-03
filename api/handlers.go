package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

type ActionResult struct {
	Message string  `json:"message,omitempty"`
	Error   string  `json:"error,omitempty"`
	Player  *Player `json:"player,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("erreur interne : %v", err)
	writeJSON(w, http.StatusInternalServerError, ActionResult{Error: "💥 Erreur interne du serveur de jeu."})
}

func parseID(r *http.Request) (uint64, error) {
	return strconv.ParseUint(r.PathValue("id"), 10, 64)
}

// act exécute une action de jeu dans une transaction avec verrou sur le joueur
// (évite les doubles clics / courses entre requêtes).
func (s *Server) act(w http.ResponseWriter, r *http.Request, fn func(tx *sql.Tx, p *Player) (string, error)) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ActionResult{Error: "Identifiant invalide."})
		return
	}

	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()

	p, err := loadPlayer(tx, id, true)
	if errors.Is(err, errNoPlayer) {
		writeJSON(w, http.StatusNotFound, ActionResult{Error: "Tu n'as pas encore de personnage. Utilise /jouer."})
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}

	regen(p)
	msg, actErr := fn(tx, p)

	var ge *GameError
	if actErr != nil && !errors.As(actErr, &ge) {
		serverError(w, actErr)
		return
	}
	if actErr == nil {
		if err := savePlayer(tx, p); err != nil {
			serverError(w, err)
			return
		}
	}

	p.Inventory, err = loadInventory(tx, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}

	p.NextLevelXP = xpNeeded(p.Level)
	if p.Energy < p.MaxEnergy {
		left := energyRegenEvery - time.Since(p.lastTick)
		p.NextEnergySec = max(0, int(left.Seconds()))
	}

	if ge != nil {
		writeJSON(w, http.StatusOK, ActionResult{Error: ge.Msg, Player: p})
		return
	}
	writeJSON(w, http.StatusOK, ActionResult{Message: msg, Player: p})
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ActionResult{Error: "Identifiant invalide."})
		return
	}

	res, err := s.db.Exec(
		`INSERT IGNORE INTO players (discord_id, last_energy_tick) VALUES (?, ?)`,
		id, time.Now().UTC())
	if err != nil {
		serverError(w, err)
		return
	}
	n, _ := res.RowsAffected()
	created := n == 1
	if created {
		if err := giveStarter(s.db, id); err != nil {
			serverError(w, err)
			return
		}
	}

	s.act(w, r, func(tx *sql.Tx, p *Player) (string, error) {
		if created {
			return "🏕️ Bienvenue, survivant ! Tu pars avec un couteau rouillé et deux rations. Bonne chance…", nil
		}
		return "👋 Content de te revoir, survivant. Que fais-tu ?", nil
	})
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	s.act(w, r, func(tx *sql.Tx, p *Player) (string, error) { return "", nil })
}

func (s *Server) handleExplore(w http.ResponseWriter, r *http.Request) { s.act(w, r, doExplore) }
func (s *Server) handleFight(w http.ResponseWriter, r *http.Request)   { s.act(w, r, doFight) }
func (s *Server) handleEat(w http.ResponseWriter, r *http.Request)     { s.act(w, r, doEat) }

type TopEntry struct {
	DiscordID uint64 `json:"discord_id,string"`
	Level     int    `json:"level"`
	XP        int    `json:"xp"`
	Gold      int    `json:"gold"`
}

func (s *Server) handleTop(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(),
		`SELECT discord_id, level, xp, gold FROM players
		 ORDER BY level DESC, xp DESC, gold DESC LIMIT 10`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()

	top := []TopEntry{}
	for rows.Next() {
		var e TopEntry
		if err := rows.Scan(&e.DiscordID, &e.Level, &e.XP, &e.Gold); err != nil {
			serverError(w, err)
			return
		}
		top = append(top, e)
	}
	writeJSON(w, http.StatusOK, map[string]any{"top": top})
}
