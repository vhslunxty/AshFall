<div align="center">

# ☄️ Ashfall

**Un RPG de survie multijoueur qui se joue entièrement dans Discord.**
Explore, combats, mange, monte en niveau, grimpe au classement, sans jamais quitter ton serveur.

![Go](https://img.shields.io/badge/API-Go-00ADD8?logo=go&logoColor=white)
![Python](https://img.shields.io/badge/Bot-Python-3776AB?logo=python&logoColor=white)
![MySQL](https://img.shields.io/badge/DB-MySQL-4479A1?logo=mysql&logoColor=white)
![Docker](https://img.shields.io/badge/Deploy-Docker-2496ED?logo=docker&logoColor=white)

</div>

---

## ✨ Fonctionnalités

- 🎮 **Jeu 100 % boutons** : un panneau interactif par joueur, mis à jour en direct
- 🗺️ **Exploration** : loot, nourriture, or, armes rares… et pièges
- ⚔️ **Combats** avec ennemis qui évoluent selon ton niveau
- 🍖 **Survie** : vie, énergie qui se régénère avec le temps, inventaire
- 📈 **Progression** : XP, niveaux, PV max qui augmentent
- 🏆 **Classement** du serveur avec `/top`
- 🔒 **Anti-triche** : toute la logique est côté API, le bot n'est qu'une interface

## 🏗️ Architecture

```
Discord  <->  Bot Python (discord.py)  <->  API Go (REST)  <->  MySQL
                 (interface)               (règles du jeu)     (données)
```

| Couche | Techno | Rôle |
|---|---|---|
| `bot/` | Python, discord.py | Slash commands, embeds, boutons |
| `api/` | Go (`net/http`) | Règles du jeu, transactions, anti-double-clic |
| `db/` | MySQL 8 | Joueurs, objets, inventaires |

Comme le bot ne contient aucune règle, on peut brancher d'autres clients sur la même API (site web PHP, jeu Unity…).

## 🚀 Installation

### Prérequis
- [Docker](https://docs.docker.com/get-docker/) avec Docker Compose
- Un bot Discord ([Developer Portal](https://discord.com/developers/applications))

### 1. Créer le bot Discord
1. **New Application** → onglet **Bot** → **Reset Token** → copie le token.
2. **OAuth2 → URL Generator** : scopes `bot` + `applications.commands`, permissions `Send Messages` et `Embed Links`.
3. Ouvre l'URL générée pour inviter le bot sur ton serveur.

### 2. Configurer
```bash
git clone https://github.com/vhslunxty/ashfall.git
cd ashfall-discord-rpg
cp .env.example .env
```
Édite `.env` : `DISCORD_TOKEN`, `GUILD_ID` (recommandé, sync instantanée des commandes) et les mots de passe MySQL.

### 3. Lancer
```bash
docker compose up --build -d
docker compose logs -f bot
```
Dans Discord, tape `/jouer`. 🎉

Arrêter : `docker compose down` (ajoute `-v` pour supprimer aussi la base).

## 🎮 Comment jouer

| Commande | Description |
|---|---|
| `/jouer` | Crée ton personnage et ouvre ton panneau |
| `/top` | Classement des meilleurs survivants |
| `/aide` | Rappel des règles |

| Bouton | Coût | Effet |
|---|---|---|
| 🗺️ Explorer | 1⚡ | Loot, or, arme rare ou piège |
| ⚔️ Combattre | 2⚡ | XP, or et butin (ou défaite…) |
| 🍖 Manger | — | Consomme de la nourriture pour récupérer des PV |
| 🎒 Inventaire | — | Affiche/masque ton sac |

L'énergie remonte de **1 point toutes les 5 minutes**. Reviens régulièrement !

## 🔌 API

| Méthode | Route | Description |
|---|---|---|
| POST | `/player/{id}/start` | Crée le joueur (idempotent) |
| GET | `/player/{id}` | État complet + inventaire |
| POST | `/player/{id}/explore` | Explorer |
| POST | `/player/{id}/fight` | Combattre |
| POST | `/player/{id}/eat` | Manger |
| GET | `/top` | Top 10 |
| GET | `/health` | Vérification |

Test sans Discord :
```bash
curl -X POST localhost:8080/player/123/start
curl -X POST localhost:8080/player/123/explore
```

> ⚠️ L'API n'a pas d'authentification : elle n'est publiée que sur `127.0.0.1`. Ne l'expose pas sur Internet telle quelle.

## 🗂️ Structure du projet

```
ashfall-discord-rpg/
├── docker-compose.yml
├── .env.example
├── db/init.sql          # tables + objets de départ
├── api/
│   ├── main.go          # démarrage + routes
│   ├── handlers.go      # endpoints HTTP, transactions
│   ├── game.go          # règles : explorer, combat, manger, niveaux
│   ├── store.go         # accès base de données
│   └── Dockerfile
└── bot/
    ├── bot.py
    ├── requirements.txt
    └── Dockerfile
```

## 🛣️ Roadmap

- [✅] Craft : fabriquer des armes avec les matériaux
- [✅] Boss de serveur : PV partagés, tout le monde attaque
- [✅] Boutique et quêtes quotidiennes
- [ ] Dashboard web en direct (PHP)
- [ ] Client Unity branché sur la même API

## 🛠️ Dépannage

- **Les commandes n'apparaissent pas** : renseigne `GUILD_ID` et vérifie le scope `applications.commands` à l'invitation.
- **« Le serveur de jeu ne répond pas »** : `docker compose logs api` ; MySQL met ~30 s au premier démarrage.
- **Modifier le schéma SQL** : `init.sql` ne s'exécute qu'à la création du volume → `docker compose down -v` puis relancer.


## 📜 Me soutenir 

https://discord.gg/kXwNpC35mz
