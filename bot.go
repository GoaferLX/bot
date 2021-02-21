package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var guildID string = "518857120981057537"
var chanID string = "544575639831969822"
var goafID string = "359734862141194251"

func main() {
	// Parse config values.
	// Default value needs removing for production.
	var token string
	flag.StringVar(&token, "Token", "", "Authentication token to commuincate with bot")
	flag.Parse()
	if token == "" {
		log.Fatal("Must provide an auth token")
	}

	// map[user that triggers alert][]users that want the alert
	var m = make(map[string][]string)
	// hardcode value for testing
	//m[goafID] = []string{goafID}

	// New discord session - authenticating with bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}

	// Signal intention to listen for voice state changes only.
	//dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildVoiceStates)
	// Register handlers to session.
	dg.AddHandler(listen(m))
	dg.AddHandler(notify(m))

	// Open websocket
	if err = dg.Open(); err != nil {
		panic(err)
	}

	// Handle graceful shutdown of program - terminate socket and bot leaves server.
	log.Print("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	dg.Close()
}

// listen to a channel and alert a specified user when another user comes online.
func listen(m map[string][]string) func(*discordgo.Session, *discordgo.VoiceStateUpdate) {
	return func(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
		// Does UserID exist in map of users that trigger alerts
		alerts, ok := m[v.UserID]
		if ok {
			// get state change details
			user, _ := s.User(v.UserID)
			ch, _ := s.Channel(v.ChannelID)
			g, _ := s.Guild(v.GuildID)
			// range over each person that requires an alert for this user
			for _, u := range alerts {
				c, err := s.UserChannelCreate(u)
				if err != nil {
					panic(err)
				}
				// Message user to let them know
				s.ChannelMessageSend(c.ID, fmt.Sprintf("%s has joined the channel %s on server %s",
					user.Username, ch.Name, g.Name))
			}
		}
	}
}

func notify(alerts map[string][]string) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore bot messages
		if m.Author.ID == s.State.User.ID {
			return
		}
		switch {

		case strings.HasPrefix(m.Content, "!alert"):
			str := strings.TrimSpace(strings.TrimPrefix(m.Content, "!alert"))
			alerts[str] = append(alerts[str], m.Author.ID)

		case strings.HasPrefix(m.Content, "!remove"):
			user := strings.TrimSpace(strings.TrimPrefix(m.Content, "!remove"))
			_, ok := alerts[user]
			if !ok {
				fmt.Println("not found in map")
			}
			for i, v := range alerts[user] {
				if v == m.Author.ID {
					alerts[user] = append(alerts[user[:i]], alerts[user[i+1:]]...)
				} else {
					fmt.Println("not found in slice")
				}
			}

		}
		fmt.Println(alerts)
	}
}
