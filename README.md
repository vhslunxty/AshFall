# 🏚️ Survival RPG — mini-jeu multijoueur dans Discord

Un jeu de survie jouable uniquement avec des boutons Discord.
Architecture en 3 couches :

```
Discord  <->  Bot Python (discord.py)  <->  API Go (REST)  <->  MySQL
                 (interface)               (règles du jeu)     (données)
```

Le bot ne contient **aucune règle** : il transmet les clics à l'API Go, qui décide de tout (anti-triche) et stocke l'état dans MySQL.

## 🚀 Lancement rapide

### 1. Créer le bot Discord
1. Va sur <https://discord.com/developers/applications> → **New Application**.
2. Onglet **Bot** → **Reset Token** → copie le token.
3. Onglet **OAuth2 → URL Generator** : coche les scopes `bot` et `applications.commands`, puis la permission `Send Messages` (et `Embed Links`). Ouvre l'URL générée pour inviter le bot sur ton serveur.
   Aucun intent privilégié n'est nécessaire.

### 2. Configurer
```bash
cp .env.example .env
# édite .env : DISCORD_TOKEN, GUILD_ID (recommandé), mots de passe MySQL
```

### 3. Démarrer
```bash
docker compose up --build -d
docker compose logs -f bot
```
Dans Discord, tape `/jouer`. 🎮

Arrêter : `docker compose down` (ajoute `-v` pour effacer aussi la base de données).

## 🎮 Commandes

| Commande | Rôle |
|---|---|
| `/jouer` | Crée ton personnage / ouvre ton panneau avec les boutons |
| `/top` | Classement des meilleurs survivants |
| `/aide` | Mode d'emploi |

Boutons : **🗺️ Explorer** (1⚡) · **⚔️ Combattre** (2⚡) · **🍖 Manger** · **🎒 Inventaire**.
L'énergie remonte de 1 point toutes les 5 minutes (constante `energyRegenEvery` dans `api/game.go`).

## 🗂️ Structure

```
survival-discord-rpg/
├── docker-compose.yml
├── .env.example
├── db/init.sql          # tables + objets de départ
├── api/                 # API Go (net/http, MySQL)
│   ├── main.go          # démarrage + routes
│   ├── handlers.go      # endpoints HTTP, transactions
│   ├── game.go          # règles : explorer, combat, manger, niveaux
│   ├── store.go         # accès base de données
│   └── Dockerfile
└── bot/                 # bot discord.py
    ├── bot.py
    ├── requirements.txt
    └── Dockerfile
```

## 🔌 Endpoints de l'API

| Méthode | Route | Description |
|---|---|---|
| POST | `/player/{id}/start` | Crée le joueur (idempotent) |
| GET | `/player/{id}` | État complet + inventaire |
| POST | `/player/{id}/explore` | Explorer (loot, or, piège…) |
| POST | `/player/{id}/fight` | Combat contre un ennemi |
| POST | `/player/{id}/eat` | Manger pour récupérer des PV |
| GET | `/top` | Top 10 |
| GET | `/health` | Vérification |

Test rapide sans Discord :
```bash
curl -X POST localhost:8080/player/123/start
curl -X POST localhost:8080/player/123/explore
```

> ⚠️ L'API n'a **pas d'authentification** : elle est publiée uniquement sur `127.0.0.1`. Ne l'expose pas sur Internet telle quelle.

## 🧩 Pistes d'évolution

- **Craft** : utiliser les matériaux (bois, ferraille…) pour fabriquer des armes (table `recipes`).
- **Boss de serveur** : un boss commun que tous les joueurs attaquent, avec PV partagés.
- **Dashboard PHP** (ton Pulse-Dashboard) branché sur `/top` ou directement sur MySQL.
- **Événements aléatoires** quotidiens, boutique, quêtes.
- Intégration **Unity** : un client de plus qui appelle la même API.

## 🛠️ Dépannage

- *Les commandes n'apparaissent pas* : renseigne `GUILD_ID` (sync instantanée) et vérifie que le bot a été invité avec le scope `applications.commands`.
- *« Le serveur de jeu ne répond pas »* : `docker compose logs api` ; au premier démarrage MySQL met ~20-30 s.
- *Changer le schéma SQL* : `init.sql` ne s'exécute qu'à la création du volume → `docker compose down -v` puis relancer.
