package config

import "github.com/Vaayne/aienvoy/pkg/llms/llm"

type Config struct {
	Service  ServiceConfig
	Admins   []Admin
	LLMs     []llm.Config
	Axiom    Axiom
	Telegram struct {
		Token string `yaml:"token"`
	}
	ClaudeWeb struct {
		Token string `yaml:"token"`
	}
	Bard struct {
		Token string `yaml:"token"`
	}
	ReadEase    ReadEase
	CookieCloud CookieCloud
	MidJourney  MidJourney
	AWS         AWSConfig
}

type ServiceConfig struct {
	Name     string
	Host     string
	Port     string
	URL      string
	LogLevel string
	Env      string
}

type Axiom struct {
	Token   string
	Dataset string
}

type Admin struct {
	Email    string
	Password string
}

type ReadEase struct {
	TelegramChannel int64 `yaml:"telegramChannel"`
	TopStoriesCnt   int   `yaml:"topStoriesCnt"`
}

type CookieCloud struct {
	Host string
	UUID string
	Pass string
}

type MidJourney struct {
	DiscordUserToken string
	DiscordBotToken  string
	DiscordServerId  string
	DiscordChannelId int64
	DiscordAppId     string
	DiscordSessionId string
}

type AWSConfig struct {
	Region          string
	AccessKeyId     string
	SecretAccessKey string
}
