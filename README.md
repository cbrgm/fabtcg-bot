# Flesh and Blood TCG Bot

***A Bot for Legend Story Studios TCG [Flesh and Blood](https://fabtcg.com/).***

## About

This bot sends information about cards from the trading card game Flesh and Blood to different channels

### Where does the data come from?
The data of this bot is provided by https://fabdb.net.

## Messengers

This code powers the bot [fabtcg_bot](https://t.me/fabtcg_bot) on Telegram. Feel free to check it out!
Right now only [Telegram](https://telegram.org) is supported, but I'd like to add more in the future.

## Installation

Container Usage:

```
docker pull quay.io/cbrgm/fabtcg-bot:latest
```

Start as a command:
```
docker run -d \
    -e 'TELEGRAM_ADMIN=1234567' \
    -e 'TELEGRAM_TOKEN=XXX' \
    --name fabtcg-bot
    quay.io/cbrgm/fabtcg-bot:latest
```

### Configuration

```
Usage: fabtcg-bot --telegram.token=STRING

Flags:
  -h, --help                                 Show context-sensitive help.
      --http.addr="0.0.0.0:8080"             The address the fabtcg-bot metrics are exposed
      --log.level="info"                     The log level to use for filtering logs
      --telegram.admin=TELEGRAM.ADMIN,...    The ID of the initial Telegram Admin
      --telegram.token=STRING                The token used to connect with Telegram ($TELEGRAM_TOKEN)
      --metrics.profile                      Enable pprof profiling
      --metrics.runtime                      Enable bot runtime metrics
      --metrics.enabled                      Enable bot metrics
      --metrics.prefix=""                    Set metrics prefix path

```

## Development
Build the binary using `make`:

```
make
```

In case you have `$GOPATH/bin` in your `$PATH` you can now start the bot by running:
```
fabtcg-bot
```

## Contributing & License

Feel free to submit changes! See
the [Contributing Guide](https://github.com/cbrgm/contributing/blob/master/CONTRIBUTING.md). This project is open-source
and is developed under the terms of
the [MIT License](https://github.com/cbrgm/fabtcg-bot/blob/master/LICENSE).

## Disclaimer
This Bot is in no way affiliated with [Legend Story Studios®](https://legendstory.com/). All intellectual IP belongs to Legend Story Studios®,
Flesh & Blood™, and set names are trademarks of Legend Story Studios®. Flesh and Blood™ characters, cards, logos,
and art are property of Legend Story Studios®.
