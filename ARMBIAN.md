# Wire‑Pod for Armbian

This project contains the Wire‑Pod sources based on upstream commit
`a04190f93491435257abc2a2d933f7190b3a6f3a`, plus the Armbian configuration,
Vector battery percentage, Chinese voice recognition, configurable
OpenAI-compatible knowledge providers, and additional weather providers.
The source retains the upstream MIT license.

## Docker setup

On an Armbian host with Docker Engine and Docker Compose, run
`docker compose up -d --build`. Open port 8080 on the host to complete setup.
The server's Docker volumes store pairing and settings between updates.
Run only one instance of Wire‑Pod on the host's ports. Keep the settings
page on a trusted network.

The default speech recognition language is `zh-CN`. To change it, set
`WIREPOD_STT_LANGUAGE` before starting Compose. The normal Wire‑Pod setup
flow downloads the selected speech model. Build steps obtain the
public upstream Escape Pod compatibility files and verify their
checksums. For a source-only build, run
`bash docker/fetch-upstream-compatibility-files.sh` before compiling.

For an existing deployment, reuse the existing Wire‑Pod volumes. Compose
usually prefixes new volume names with the project name. Use external
volumes explicitly when reusing an installation with different names.

The built-in Mandarin speech path uses eSpeak NG as a fallback. To reuse
a Piper installation, mount its local directory at
`/opt/wirepod-mandarin:ro` with an executable `synthesize` wrapper that
reads UTF-8 text on stdin and writes WAV on stdout.
`docker/synthesize-mandarin.sh` is a sample wrapper for the zh_CN-huayan
voice. Piper binaries, voice models and existing runtime data are
installed locally.

## Weather settings

Open **Weather API Setup** to choose Open-Meteo, OpenWeatherMap,
WeatherAPI or **Custom JSON API**. Open-Meteo needs no API key. Set the
default city in **Bot Settings**. The custom provider can also override
the city and temperature unit.

The custom provider sends an HTTP GET request. In the query string you
can use `{location}`, `{unit}` (C or F) and `{hours}` (0 for current
conditions). Configure the API to return temperatures in the selected
unit. For authenticated services, use the dedicated password field and
set either the header name or the query parameter name. For a bearer
token, enter the full header value including the `Bearer ` prefix.

Suppose an API returns:

```json
{"current":{"temp":12.4,"weather":61}}
```

Set the temperature path to `/current/temp`, the condition path to
`/current/weather`, and the mapping to `{"61":"Rain"}`. Numeric paths
such as `/forecast/0/temp` can read array entries. The default paths
are `/temperature` and `/condition`.

The weather response goes to Vector's firmware, which plays the
original weather voice and animation. Supported condition names
include Sunny, Clear, Stars, Cloudy, Rain, Snow, Thunderstorms, Windy
and Cold.

Enter model and weather keys on the server. Keep API configuration,
robot pairing files, backups and downloaded models outside the
repository.
