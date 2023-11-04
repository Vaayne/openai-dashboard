package main

import (
	"embed"
	"log/slog"

	"github.com/Vaayne/aienvoy/internal/core/midjourney"
	_ "github.com/Vaayne/aienvoy/internal/pkg/logger"
	"github.com/Vaayne/aienvoy/internal/ports/httpserver"
	"github.com/Vaayne/aienvoy/internal/ports/tgbot"
	_ "github.com/Vaayne/aienvoy/migrations"

	"github.com/pocketbase/pocketbase/tools/cron"

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
	// migrate DB
	migratecmd.MustRegister(app, app.RootCmd, &migratecmd.Options{
		Automigrate: false,
	})

	// scheduled jobs
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		scheduler := cron.New()
		// every 5 minutes to run readease job
		// if config.GetConfig().ReadEase.TelegramChannel != 0 {
		// 	scheduler.MustAdd("readease", "0 * * * *", func() {
		// 		summaries, err := readease.PeriodJob(app)
		// 		if err != nil {
		// 			slog.Error("run period readease job error", "err", err)
		// 		}
		// 		bot := tgbot.DefaultBot(app)
		// 		channel := tb.ChatID(config.GetConfig().ReadEase.TelegramChannel)
		// 		for _, summary := range summaries {
		// 			msg, err := bot.Send(channel, summary)
		// 			if err != nil {
		// 				slog.Error("failed to send readease message to channel", "err", err, "msg", msg)
		// 			}
		// 		}
		// 	})
		// }
		scheduler.Start()
		return nil
	})
	// register routes
	registerRoutes(app)

	// after bootstrap start other service
	app.OnAfterBootstrap().Add(func(e *core.BootstrapEvent) error {
		// start telegram bot
		go tgbot.Serve(app)
		// start midjourney bot
		go func() {
			m := midjourney.New(app.Dao())
			m.Client.Serve()
		}()
		return nil
	})

	if err := app.Start(); err != nil {
		slog.Error("failed to start app", "err", err)
	}
}
