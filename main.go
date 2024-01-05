package main

import (
	"fmt"
	"gamemediabot/botmanager"
	"gamemediabot/xapi"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	discordToken := "Bot " + os.Getenv("DISCORD_TOKEN")
	discordClinetID := os.Getenv("DISCORD_CLIENT_ID")
	xBearer := os.Getenv("X_BEARER_TOKEN")
	initialBatchDurationMinu := 5

	fmt.Println("discordToken:", discordToken)
	fmt.Println("discordClinetID:", discordClinetID)
	fmt.Println("xBearer:", xBearer)

	client := xapi.NewXClinet(xBearer)
	discord, err := discordgo.New(discordToken)
	discord.Token = discordToken
	if err != nil {
		log.Fatal("Error creating Discord session: ", err)
	}

	bot := botmanager.NewBotManager(discord, client, initialBatchDurationMinu)
	botmanager.SetGlobalManager(bot)

	err = discord.Open()
	if err != nil {
		log.Fatal("Error opening connection: ", err)
	}
	defer discord.Close()
	bot.Start()

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	stopBot := make(chan os.Signal, 1)
	signal.Notify(stopBot, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-stopBot
}
