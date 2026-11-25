var botStats = document.getElementById("botStats");
const botStatusPollMs = 5000;
const batteryPollMs = 30000;
const botChips = {};

const holidayLightGifs = [
  "faceholidaylights.gif",
  "faceholidaylights1.gif",
  "faceholidaylights2.gif",
  "faceholidaylights3.gif"
];
const holidayGif = holidayLightGifs[Math.floor(Math.random() * holidayLightGifs.length)];

function activeFace() {
  const now = new Date();
  const month = now.getMonth();
  const day = now.getDate();
  if ((month == 11 && day == 31) || (month == 0 && day == 1)) {
    return "facehnyfirework.gif";
  }
  if (month == 11 && (day == 24 || day == 25)) {
    return holidayGif;
  }
  if (month == 0 || month == 11) {
    return "facesnowglobe.gif";
  }
  return "face.gif";
}

function lastSeenText(secs) {
  if (secs < 0) {
    return "not seen yet";
  }
  if (secs < 60) {
    return "seen " + secs + "s ago";
  }
  if (secs < 3600) {
    return "seen " + Math.floor(secs / 60) + "m ago";
  }
  if (secs < 86400) {
    return "seen " + Math.floor(secs / 3600) + "h ago";
  }
  return "seen " + Math.floor(secs / 86400) + "d ago";
}

function makeBotChip(bot) {
  const chip = document.createElement("div");
  chip.className = "botChip";
  chip.onclick = function() {
    if (!chip.classList.contains("active")) {
      return;
    }
    window.location.href = "/sdkapp/settings.html?serial=" + encodeURIComponent(bot.esn);
  };

  const face = document.createElement("div");
  face.className = "vectorFace";
  chip.appendChild(face);

  const text = document.createElement("div");
  text.className = "botChipText";
  const name = document.createElement("span");
  name.className = "botChipName";
  const statusText = document.createElement("span");
  statusText.className = "botChipStatus";
  text.appendChild(name);
  text.appendChild(statusText);
  chip.appendChild(text);

  botStats.appendChild(chip);
  return { chip, face, name, statusText, face_url: "", batteryText: "", nextBatteryPoll: 0, batteryPending: false };
}

function estimateBatteryPercent(voltage) {
  // Match the voltage curve used by the older Wire-Pod battery card.
  if (!Number.isFinite(voltage) || voltage <= 0) return null;
  if (voltage >= 4.10) return 100;
  if (voltage <= 3.50) return 0;
  if (voltage >= 3.85) {
    return Math.round(80 + 20 * Math.log10(1 + 9 * (voltage - 3.85) / 0.25));
  }
  return Math.round(80 * Math.log10(1 + 9 * (voltage - 3.50) / 0.35));
}

function formatBotBattery(data) {
  let percent = estimateBatteryPercent(data.battery_volts);
  if (percent !== null && data.battery_level === 1 && !data.is_on_charger_platform) {
    percent = Math.min(percent, 15);
  }
  let label = data.is_charging ? "Charging"
    : data.battery_level === 3 ? "Fully charged"
    : data.battery_level === 1 ? "Low battery"
    : data.is_on_charger_platform ? "On charger"
    : data.battery_level === 2 || percent !== null ? "Battery"
    : "Battery unavailable";
  if (percent === null) return label;
  return "≈" + percent + "% · " + label + " · " + data.battery_volts.toFixed(2) + " V";
}

async function refreshBotBattery(c, serial) {
  c.batteryPending = true;
  c.nextBatteryPoll = Date.now() + batteryPollMs;
  try {
    const response = await fetch("/api-sdk/get_battery?serial=" + encodeURIComponent(serial), {
      cache: "no-store",
      signal: AbortSignal.timeout(8000)
    });
    if (!response.ok) throw new Error("battery HTTP " + response.status);
    c.batteryText = formatBotBattery(await response.json());
  } catch {
    c.batteryText = "Battery unavailable";
  } finally {
    c.batteryPending = false;
    if (c.chip.classList.contains("active")) c.statusText.textContent = c.batteryText;
  }
}

function updateBotChip(c, bot) {
  c.name.textContent = bot.name || bot.esn;
  c.chip.title = "ESN: " + bot.esn + (bot.ip_address ? "\nIP: " + bot.ip_address : "");
  c.chip.classList.toggle("active", bot.active);
  if (bot.active) {
    c.statusText.textContent = c.batteryText || "active";
    if (!c.batteryPending && Date.now() >= c.nextBatteryPoll) refreshBotBattery(c, bot.esn);
  } else {
    c.statusText.textContent = lastSeenText(bot.seconds_since_seen);
    c.batteryText = "";
    c.nextBatteryPoll = 0;
  }
  const faceUrl = "url(/assets/" + (bot.active ? activeFace() : "wififace.gif") + ")";
  if (c.face_url !== faceUrl) {
    c.face.style.backgroundImage = faceUrl;
    c.face_url = faceUrl;
  }
}

async function fetchActiveBots() {
  const response = await fetch("/api-sdk/get_active_bots", {
    signal: AbortSignal.timeout(botStatusPollMs)
  });
  if (!response.ok) {
    throw new Error("bad response: " + response.status);
  }
  return await response.json();
}

async function renderBotStatus(onlySerial) {
  if (!botStats) {
    return;
  }
  const loader = document.createElement("div");
  loader.className = "botLoader";
  botStats.appendChild(loader);

  async function poll() {
    try {
      let bots = await fetchActiveBots();
      if (onlySerial) {
        bots = bots.filter((b) => b.esn.toLowerCase() === onlySerial.toLowerCase());
      }
      loader.remove();
      if (bots.length === 0) {
        botStats.style.display = "none";
      } else {
        botStats.style.display = "";
      }
      const seen = new Set();
      for (const bot of bots) {
        seen.add(bot.esn);
        if (!botChips[bot.esn]) {
          botChips[bot.esn] = makeBotChip(bot);
        }
        updateBotChip(botChips[bot.esn], bot);
      }
      for (const esn of Object.keys(botChips)) {
        if (!seen.has(esn)) {
          botChips[esn].chip.remove();
          delete botChips[esn];
        }
      }
    } catch (error) {
      console.error("Unable to get bot status:", error);
    }
    setTimeout(poll, botStatusPollMs);
  }
  poll();
}

function processBotStats() {
  renderBotStatus();
}
