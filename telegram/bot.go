package telegram

import (
	"context"
	"fmt"
	"github.com/cbrgm/fabtcg-bot/fabdb"
	"github.com/cbrgm/fabtcg-bot/metrics"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"gopkg.in/tucnak/telebot.v2"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// default
	CmdStart = "/start"
	CmdStop  = "/stop"
	CmdHelp  = "/help"
	CmdAbout = "/about"

	// debug
	CmdID = "/id"
)

const (
	responseStart = "Hi, %s! 👋 Check out " + CmdHelp + " for further details.\n You can share card information from everywhere by simply typing @fabtcg_bot followed by a card query in your chat window.\nData is provided by https://fabdb.net."
	responseStop  = "Alright, %s! I won't talk to you again 🙊. Check out " + CmdHelp + " for further details."
	responseHelp  = `
I'm a Flesh and Blood TCG Bot 🤖 on steroids for Telegram. I will send you card information directly into your telegram channels!
You can find out more about me using ` + CmdAbout + `

You can share card information from everywhere by simply typing @fabtcg_bot followed by a card query in your chat window.
	
👇 Available commands:
` + CmdStart + ` - Say hello!
` + CmdStop + ` - Say Goodbye!'.
` + CmdID + ` - Sends you your Telegram ID (works for all users!).
`
	responseAbout = `
This Telegram Bot is a non-commercial hobby project by @cbrgm and is developed as open source software for fans of the FaB TCG!

Feedback of any kind is very welcome and can be given in the Github repository at https://github.com/cbrgm/fabtcg-bot

The data of this bot is provided by https://fabdb.net. 

This Bot is in no way affiliated with Legend Story Studios®. All intellectual IP belongs to Legend Story Studios®, 
Flesh & Blood™, and set names are trademarks of Legend Story Studios®. Flesh and Blood™ characters, cards, logos, 
and art are property of Legend Story Studios®.
`
)

type Cards interface {
	ListCards(ctx context.Context, query string) ([]fabdb.Card, error)
	GetCard(ctx context.Context, identifier string) (fabdb.Card, error)
}

type Telebot interface {
	Start()
	Stop()
	Send(to telebot.Recipient, what interface{}, options ...interface{}) (*telebot.Message, error)
	Answer(query *telebot.Query, resp *telebot.QueryResponse) error
	Handle(endpoint interface{}, handler interface{})
}

type BotMetrics interface {
	IncTelegramCommands(cmd string)
	IncTelegramEventsIncoming(eventType string)
	IncTelegramEventsOutgoing(eventType string)
	RegisterHandler(path string, handler *http.ServeMux)
}

// Bot represents the telegram bot
type Bot struct {
	logger    log.Logger
	startTime time.Time
	revision  string
	cards     Cards
	metrics   BotMetrics
	telegram  Telebot

	allowlist []int
}

// BotOption passed to NewBot to change the default instance.
type BotOption func(b *Bot) error

func NewBot(state Cards, token string, opts ...BotOption) (*Bot, error) {
	poller := &telebot.LongPoller{
		Timeout: 10 * time.Second,
	}
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: poller,
	})
	if err != nil {
		return nil, err
	}
	prom := metrics.NewDefaultPrometheus()
	return NewBotWithTelegram(state, bot, prom, opts...)
}

func NewBotWithTelegram(botState Cards, bot Telebot, botMetrics BotMetrics, opts ...BotOption) (*Bot, error) {
	b := &Bot{
		logger:    log.NewNopLogger(),
		startTime: time.Now(),
		revision:  "",
		cards:     botState,
		metrics:   botMetrics,
		telegram:  bot,

		allowlist: []int{},
	}

	for _, opt := range opts {
		if err := opt(b); err != nil {
			return nil, err
		}
	}

	return b, nil
}

// WithLogger sets the logger for the Bot as an option.
func WithLogger(l log.Logger) BotOption {
	return func(b *Bot) error {
		b.logger = l
		return nil
	}
}

func WithAllowlist(ids ...int) BotOption {
	return func(b *Bot) error {
		b.allowlist = append(b.allowlist, ids...)
		sort.Ints(b.allowlist)
		return nil
	}
}

func WithMetrics(m BotMetrics) BotOption {
	return func(b *Bot) error {
		b.metrics = m
		return nil
	}
}

func WithStartTime(t time.Time) BotOption {
	return func(b *Bot) error {
		b.startTime = t
		return nil
	}
}

func WithRevision(s string) BotOption {
	return func(b *Bot) error {
		b.revision = s
		return nil
	}
}

func (b *Bot) middleware(next func(*telebot.Message) error) func(*telebot.Message) {
	return func(m *telebot.Message) {
		b.metrics.IncTelegramEventsIncoming(metrics.TelegramMessageEventType)

		if m.IsService() || m.Sender.IsBot {
			return
		}

		if !b.isOnAllowlist(int(m.Sender.ID)) && m.Text != CmdID {
			level.Info(b.logger).Log(
				"msg", "received message from forbidden sender",
				"sender_id", m.Sender.ID,
				"sender_username", m.Sender.Username,
			)
			return
		}

		command := strings.Split(m.Text, " ")[0]
		b.metrics.IncTelegramCommands(command)

		level.Debug(b.logger).Log("msg", "received message", "text", m.Text)
		err := next(m)
		if err != nil {
			level.Warn(b.logger).Log("msg", "failed to handle bot command", "err", err)
			return
		}

		b.metrics.IncTelegramEventsOutgoing(metrics.TelegramMessageEventType)
	}
}

func (b *Bot) queryMiddleware(next func(query *telebot.Query) error) func(query *telebot.Query) {
	return func(m *telebot.Query) {
		b.metrics.IncTelegramEventsIncoming(metrics.TelegramInlineQueryEventType)

		if m.From.IsBot || len(m.Text) <= 3 {
			return
		}

		if !b.isOnAllowlist(int(m.From.ID)) && m.Text != CmdID {
			level.Info(b.logger).Log(
				"msg", "received message from forbidden sender",
				"sender_id", m.From.ID,
				"sender_username", m.From.Username,
			)
			return
		}

		level.Debug(b.logger).Log("msg", "received message", "text", m.Text)

		err := next(m)
		if err != nil {
			level.Warn(b.logger).Log("msg", "failed to handle inline query", "err", err)
			return
		}

		b.metrics.IncTelegramEventsOutgoing(metrics.TelegramInlineQueryEventType)
	}
}

// isOnAllowlist checks whether the id of a telegram user is listed on the allowlist
// returns true if Bot.allowlist is empty (e.g. all users are allowed) or the id was found on the list
func (b *Bot) isOnAllowlist(id int) bool {
	if len(b.allowlist) == 0 {
		return true
	}
	i := sort.SearchInts(b.allowlist, id)
	return i < len(b.allowlist) && b.allowlist[i] == id
}

// Run runs the but, starting all goroutines
func (b *Bot) Run(ctx context.Context) error {
	b.telegram.Handle(CmdStart, b.middleware(b.handleStart))
	b.telegram.Handle(CmdStop, b.middleware(b.handleStop))
	b.telegram.Handle(CmdHelp, b.middleware(b.handleHelp))
	b.telegram.Handle(CmdAbout, b.middleware(b.handleAbout))
	b.telegram.Handle(CmdID, b.middleware(b.handleID))

	// handle inline commands
	b.telegram.Handle(telebot.OnQuery, b.queryMiddleware(b.handleOnQuery))

	var gr run.Group
	{
		gr.Add(func() error {
			b.telegram.Start()
			return nil
		}, func(err error) {
			b.telegram.Stop()
		})
	}
	return gr.Run()
}

func (b *Bot) handleStart(message *telebot.Message) error {
	level.Info(b.logger).Log(
		"msg", "user executed start command",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)

	_, err := b.telegram.Send(message.Sender, fmt.Sprintf(responseStart, message.Sender.FirstName))
	return err
}

func (b *Bot) handleStop(message *telebot.Message) error {
	level.Info(b.logger).Log(
		"msg", "user executed stop command",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)

	_, err := b.telegram.Send(message.Sender, fmt.Sprintf(responseStop, message.Sender.FirstName))
	return err
}

func (b *Bot) handleHelp(message *telebot.Message) error {
	level.Info(b.logger).Log(
		"msg", "user executed help command",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)
	_, err := b.telegram.Send(message.Chat, responseHelp)
	return err
}

func (b *Bot) handleAbout(message *telebot.Message) error {
	level.Info(b.logger).Log(
		"msg", "user executed about command",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)
	_, err := b.telegram.Send(message.Chat, responseAbout)
	return err
}

func (b *Bot) handleOnQuery(q *telebot.Query) error {
	cards, err := b.cards.ListCards(context.Background(), q.Text)
	if err != nil {
		level.Warn(b.logger).Log(
			"msg", "failed to query cards",
			"from", q.From.ID,
			"query", q.Text,
			"err", err,
		)
		return err
	}

	results := make(telebot.Results, len(cards))
	for i, card := range cards {
		result := &telebot.PhotoResult{
			URL:         card.Image,
			Title:       card.Name,
			Description: card.Text,
			ThumbURL:    card.Image,
		}

		results[i] = result
		results[i].SetResultID(strconv.Itoa(i))
	}

	err = b.telegram.Answer(q, &telebot.QueryResponse{
		Results:   results,
		CacheTime: 60,
	})
	if err != nil {
		level.Warn(b.logger).Log(
			"msg", "failed to send query response",
			"from", q.From.ID,
			"query", q.Text,
			"err", err,
		)
		return err
	}
	return err
}

func (b *Bot) handleID(message *telebot.Message) error {
	level.Info(b.logger).Log(
		"msg", "user executed id command",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)

	if message.Private() {
		_, err := b.telegram.Send(message.Chat, fmt.Sprintf("Your user id is %d", message.Sender.ID))
		return err
	}
	return nil
}
