package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"twitch-obs-bot/auth"
	"twitch-obs-bot/automation"
	automationactions "twitch-obs-bot/automation/actions"
	"twitch-obs-bot/events"
	"twitch-obs-bot/logging"

	"github.com/andreykaipov/goobs"
	twitch "github.com/gempir/go-twitch-irc/v4"
)

func main() {
	logger := logging.New(os.Getenv("LOG_LEVEL"))
	systemLog := logger.With("component", "system")
	authLog := logger.With("component", "auth")
	obsLog := logger.With("component", "obs")
	engineLog := logger.With("component", "engine")
	twitchLog := logger.With("component", "twitch")
	ingestLog := logger.With("component", "ingest")

	// --------------------------------------------------
	// Environment
	// --------------------------------------------------

	twitchUser := os.Getenv("TWITCH_USERNAME")
	channel := os.Getenv("TWITCH_CHANNEL")
	obsPassword := os.Getenv("OBS_PASSWORD")
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	clientSecret := os.Getenv("TWITCH_CLIENT_SECRET")

	if twitchUser == "" || channel == "" || clientID == "" || clientSecret == "" {
		fatal(
			systemLog,
			"missing required environment variables",
			slog.String("missing", missingEnvVars(map[string]string{
				"TWITCH_USERNAME":      twitchUser,
				"TWITCH_CHANNEL":       channel,
				"TWITCH_CLIENT_ID":     clientID,
				"TWITCH_CLIENT_SECRET": clientSecret,
			})),
		)
	}

	// --------------------------------------------------
	// Root Context + Signal Handling
	// --------------------------------------------------

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// --------------------------------------------------
	// OAuth
	// --------------------------------------------------

	manager := &auth.Manager{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		File:         "token.json",
		Logger:       authLog,
	}

	if _, err := os.Stat("token.json"); os.IsNotExist(err) {
		authLog.Info("no token found, starting oauth flow")
		if err := manager.Authorize(twitchUser); err != nil {
			fatal(authLog, "authorization failed", slog.Any("error", err))
		}
		authLog.Info("authorization complete")
	}

	accessToken, err := manager.GetValidAccessToken()
	if err != nil {
		fatal(authLog, "token error", slog.Any("error", err))
	}

	// --------------------------------------------------
	// OBS
	// --------------------------------------------------

	obs, err := goobs.New("localhost:4455", goobs.WithPassword(obsPassword))
	if err != nil {
		fatal(obsLog, "connection failed", slog.Any("error", err))
	}
	obsLog.Info("connected")

	// --------------------------------------------------
	// Event Bus
	// --------------------------------------------------

	bus := events.NewBus(logger.With("component", "bus"))

	// --------------------------------------------------
	// Load Automation Config
	// --------------------------------------------------

	cfg, err := automation.Load("automations.json")
	if err != nil {
		fatal(engineLog, "failed to load automation config",
			slog.String("file", "automations.json"),
			slog.Any("error", err),
		)
	}

	// --------------------------------------------------
	// Action Registry + Engine
	// --------------------------------------------------

	registry := automation.NewActionRegistry()

	engine := automation.NewEngine(cfg, registry, ctx, logger)
	vars := engine.Vars()

	registry.Register("switch_scene", &automationactions.SwitchSceneAction{
		Obs: obs,
	})

	registry.Register("set_var", &automationactions.SetVarAction{
		Vars: vars,
	})

	registry.Register("wait", &automationactions.WaitAction{})

	engine.LogCapabilities()

	// --------------------------------------------------
	// Subscribe Engine to Events
	// --------------------------------------------------

	bus.Subscribe("chat.message", func(e events.Event) {
		go engine.HandleEvent(e)
	})

	// --------------------------------------------------
	// Twitch Client
	// --------------------------------------------------

	client := twitch.NewClient(twitchUser, "oauth:"+accessToken)
	engine.SetNotifier(&automation.ChatNotifier{
		Client:  client,
		Channel: channel,
	})

	client.OnConnect(func() {
		twitchLog.Info("connected")
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if !(message.User.Badges["broadcaster"] == 1 ||
			message.User.Badges["moderator"] == 1) {
			return
		}

		ingestLog.Info(
			"event received",
			slog.String("event", "chat.message"),
			slog.String("user", message.User.Name),
			slog.String("message", message.Message),
		)

		bus.Publish(events.Event{
			Type: "chat.message",
			Data: map[string]any{
				"user":    message.User.Name,
				"message": message.Message,
			},
		})
	})

	client.Join(channel)

	// --------------------------------------------------
	// Run Twitch Client
	// --------------------------------------------------

	go func() {
		if err := client.Connect(); err != nil {
			twitchLog.Error("connection closed", slog.Any("error", err))
		}
	}()

	// --------------------------------------------------
	// Graceful Shutdown
	// --------------------------------------------------

	go func() {
		<-sig
		systemLog.Info("shutdown signal received")

		cancel()            // cancel engine executions
		client.Disconnect() // stop twitch
		obs.Disconnect()    // close OBS
	}()

	// Block until context is cancelled
	<-ctx.Done()

	systemLog.Info("shutdown complete")
}

func fatal(logger *slog.Logger, msg string, args ...any) {
	logger.Error(msg, args...)
	os.Exit(1)
}

func missingEnvVars(values map[string]string) string {
	var missing []string
	for key, value := range values {
		if value == "" {
			missing = append(missing, key)
		}
	}

	sort.Strings(missing)
	return strings.Join(missing, ",")
}
