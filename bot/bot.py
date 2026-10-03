import asyncio
import logging
import os

import aiohttp
import discord
from discord import app_commands

TOKEN = os.environ["DISCORD_TOKEN"]
API_URL = os.getenv("API_URL", "http://api:8080").rstrip("/")
GUILD_ID = os.getenv("GUILD_ID", "").strip()

logging.basicConfig(level=logging.INFO)
log = logging.getLogger("survival-bot")


def bar(cur: int, mx: int, size: int = 10) -> str:
    mx = max(mx, 1)
    filled = round(size * max(0, min(cur, mx)) / mx)
    return "█" * filled + "░" * (size - filled)


class SurvivalBot(discord.Client):
    def __init__(self) -> None:
        super().__init__(intents=discord.Intents.default())
        self.tree = app_commands.CommandTree(self)
        self.session: aiohttp.ClientSession | None = None

    async def setup_hook(self) -> None:
        self.session = aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=10))
        if GUILD_ID:
            guild = discord.Object(id=int(GUILD_ID))
            self.tree.copy_global_to(guild=guild)
            await self.tree.sync(guild=guild)
            log.info("Commandes synchronisées sur le serveur %s", GUILD_ID)
        else:
            await self.tree.sync()
            log.info("Commandes synchronisées globalement (peut prendre un moment)")

    async def close(self) -> None:
        if self.session:
            await self.session.close()
        await super().close()


bot = SurvivalBot()


async def api(method: str, path: str) -> dict:
    """Appelle l'API Go et renvoie toujours un dict (jamais d'exception)."""
    try:
        async with bot.session.request(method, f"{API_URL}{path}") as resp:
            return await resp.json()
    except (aiohttp.ClientError, asyncio.TimeoutError):
        log.exception("Appel API échoué : %s %s", method, path)
        return {"error": "🔌 Le serveur de jeu ne répond pas, réessaie dans un instant."}


def build_embed(user: discord.abc.User, data: dict, show_inventory: bool) -> discord.Embed:
    p = data["player"]
    is_error = bool(data.get("error"))
    text = data.get("error") or data.get("message") or "Que veux-tu faire ?"

    embed = discord.Embed(
        title=f"🏚️ {user.display_name} — Niveau {p['level']}",
        description=text,
        color=discord.Color.red() if is_error else discord.Color.green(),
    )
    embed.add_field(
        name="❤️ Vie",
        value=f"`{bar(p['hp'], p['max_hp'])}` {p['hp']}/{p['max_hp']}",
        inline=False,
    )

    energy = f"`{bar(p['energy'], p['max_energy'])}` {p['energy']}/{p['max_energy']}"
    if p["energy"] < p["max_energy"]:
        sec = p.get("next_energy_sec", 0)
        energy += f"  (+1 dans {sec // 60}m{sec % 60:02d}s)"
    embed.add_field(name="⚡ Énergie", value=energy, inline=False)

    embed.add_field(
        name="⭐ XP",
        value=f"`{bar(p['xp'], p['next_level_xp'])}` {p['xp']}/{p['next_level_xp']}",
        inline=False,
    )
    embed.add_field(name="💰 Or", value=str(p["gold"]), inline=True)

    if show_inventory:
        lines = []
        for it in p.get("inventory", []):
            extra = f" (⚔️ +{it['power']})" if it["type"] == "weapon" else ""
            extra = f" (❤️ +{it['power']})" if it["type"] == "food" else extra
            lines.append(f"{it['emoji']} **{it['name']}** x{it['quantity']}{extra}")
        value = "\n".join(lines) or "*Vide*"
        embed.add_field(name="🎒 Inventaire", value=value[:1000], inline=False)

    embed.set_footer(text="Explorer: 1⚡ • Combattre: 2⚡")
    return embed


class GameView(discord.ui.View):
    def __init__(self, owner: discord.abc.User) -> None:
        super().__init__(timeout=900)
        self.owner = owner
        self.show_inventory = False
        self.message: discord.Message | None = None

    async def interaction_check(self, interaction: discord.Interaction) -> bool:
        if interaction.user.id != self.owner.id:
            await interaction.response.send_message(
                "Ce n'est pas ton personnage ! Fais `/jouer` pour avoir le tien.",
                ephemeral=True,
            )
            return False
        return True

    async def on_timeout(self) -> None:
        for child in self.children:
            child.disabled = True
        if self.message:
            try:
                await self.message.edit(view=self)
            except discord.HTTPException:
                pass

    async def render(self, interaction: discord.Interaction, data: dict) -> None:
        if "player" not in data:
            await interaction.response.send_message(
                data.get("error", "Erreur inconnue."), ephemeral=True
            )
            return
        await interaction.response.edit_message(
            embed=build_embed(self.owner, data, self.show_inventory), view=self
        )

    @discord.ui.button(label="Explorer", emoji="🗺️", style=discord.ButtonStyle.primary)
    async def explore(self, interaction: discord.Interaction, button: discord.ui.Button):
        await self.render(interaction, await api("POST", f"/player/{interaction.user.id}/explore"))

    @discord.ui.button(label="Combattre", emoji="⚔️", style=discord.ButtonStyle.danger)
    async def fight(self, interaction: discord.Interaction, button: discord.ui.Button):
        await self.render(interaction, await api("POST", f"/player/{interaction.user.id}/fight"))

    @discord.ui.button(label="Manger", emoji="🍖", style=discord.ButtonStyle.success)
    async def eat(self, interaction: discord.Interaction, button: discord.ui.Button):
        await self.render(interaction, await api("POST", f"/player/{interaction.user.id}/eat"))

    @discord.ui.button(label="Inventaire", emoji="🎒", style=discord.ButtonStyle.secondary)
    async def inventory(self, interaction: discord.Interaction, button: discord.ui.Button):
        self.show_inventory = not self.show_inventory
        await self.render(interaction, await api("GET", f"/player/{interaction.user.id}"))


@bot.tree.command(name="jouer", description="Ouvre ton panneau de survie")
async def jouer(interaction: discord.Interaction):
    data = await api("POST", f"/player/{interaction.user.id}/start")
    if "player" not in data:
        await interaction.response.send_message(
            data.get("error", "Erreur inconnue."), ephemeral=True
        )
        return
    view = GameView(interaction.user)
    await interaction.response.send_message(
        embed=build_embed(interaction.user, data, False), view=view
    )
    view.message = await interaction.original_response()


@bot.tree.command(name="top", description="Classement des meilleurs survivants")
async def top(interaction: discord.Interaction):
    data = await api("GET", "/top")
    entries = data.get("top")
    if entries is None:
        await interaction.response.send_message(
            data.get("error", "Erreur inconnue."), ephemeral=True
        )
        return
    if not entries:
        await interaction.response.send_message("Personne n'a encore joué. Fais `/jouer` !")
        return

    medals = ["🥇", "🥈", "🥉"]
    lines = []
    for i, e in enumerate(entries):
        rank = medals[i] if i < 3 else f"`{i + 1}.`"
        lines.append(f"{rank} <@{e['discord_id']}> — Niv. **{e['level']}** • {e['xp']} XP • 💰 {e['gold']}")
    embed = discord.Embed(
        title="🏆 Meilleurs survivants",
        description="\n".join(lines),
        color=discord.Color.gold(),
    )
    await interaction.response.send_message(embed=embed)


@bot.tree.command(name="aide", description="Comment jouer")
async def aide(interaction: discord.Interaction):
    await interaction.response.send_message(
        "**🏚️ Survie — mode d'emploi**\n"
        "• `/jouer` : ouvre ton panneau (crée ton personnage la 1re fois)\n"
        "• 🗺️ **Explorer** (1⚡) : loot, or, pièges…\n"
        "• ⚔️ **Combattre** (2⚡) : XP, or et butin\n"
        "• 🍖 **Manger** : consomme de la nourriture pour te soigner\n"
        "• 🎒 **Inventaire** : affiche/masque ton sac\n"
        "• `/top` : classement du serveur\n\n"
        "L'énergie remonte de 1 point toutes les 5 minutes.",
        ephemeral=True,
    )


@bot.event
async def on_ready():
    log.info("Connecté en tant que %s ✅", bot.user)


if __name__ == "__main__":
    bot.run(TOKEN)
