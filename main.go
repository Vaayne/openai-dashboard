package main

import (
	"embed"
	"log/slog"

	"aienvoy/internal/core/readease"
	"aienvoy/internal/pkg/config"
	"aienvoy/internal/pkg/logger"
	"aienvoy/internal/ports/httpserver"
	"aienvoy/internal/ports/tgbot"
	_ "aienvoy/migrations"

	"github.com/pocketbase/pocketbase/tools/cron"

	tb "gopkg.in/telebot.v3"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

//go:embed all:web
var staticFiles embed.FS

func registerRoutes(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		httpserver.RegisterRoutes(e.Router, app, staticFiles)
		return nil
	})
}

func main() {
	app := pocketbase.New()

	go tgbot.Serve(app)

	migratecmd.MustRegister(app, app.RootCmd, &migratecmd.Options{
		Automigrate: false,
	})

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		scheduler := cron.New()
		// every 5 minutes to run readease job
		scheduler.MustAdd("readease", "0 * * * *", func() {
			summaries, err := readease.ReadEasePeriodJob(app)
			if err != nil {
				slog.Error("run period readease job error", "err", err)
			}
			bot := tgbot.DefaultBot(app)
			channel := tb.ChatID(config.GetConfig().Telegram.ReadeaseChannel)
			for _, summary := range summaries {
				msg, err := bot.Send(channel, summary)
				if err != nil {
					slog.Error("failed to send readease message to channel", "err", err, "msg", msg)
				}
			}
		})
		scheduler.Start()
		return nil
	})

	registerRoutes(app)
	if err := app.Start(); err != nil {
		logger.SugaredLogger.Fatalw("failed to start app", "error", err)
	}
}
