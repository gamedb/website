package chatbot

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/gamedb/gamedb/pkg/helpers"
	"github.com/gamedb/gamedb/pkg/helpers/memcache"
	"github.com/gamedb/gamedb/pkg/log"
	"github.com/gamedb/gamedb/pkg/mongo"
	"github.com/gamedb/gamedb/pkg/queue"
	"go.mongodb.org/mongo-driver/bson"
)

type CommandPlayerLevel struct {
}

func (CommandPlayerLevel) Regex() string {
	return `^[.|!]level (.{2,32})$`
}

func (CommandPlayerLevel) DisableCache() bool {
	return false
}

func (CommandPlayerLevel) Example() string {
	return ".level PlayerName"
}

func (CommandPlayerLevel) Description() string {
	return "Get the level of a player"
}

func (CommandPlayerLevel) Type() CommandType {
	return TypePlayer
}

func (c CommandPlayerLevel) Output(msg *discordgo.MessageCreate) (message discordgo.MessageSend, err error) {

	matches := RegexCache[c.Regex()].FindStringSubmatch(msg.Message.Content)

	player, q, err := mongo.SearchPlayer(matches[1], bson.M{"_id": 1, "persona_name": 1, "level": 1})
	if err == mongo.ErrNoDocuments {

		message.Content = "Player **" + matches[1] + "** not found, please enter a user's vanity URL"
		return message, nil

	} else if err != nil {
		return message, err
	}

	if q {
		err = queue.ProducePlayer(queue.PlayerMessage{ID: player.ID})
		err = helpers.IgnoreErrors(err, memcache.ErrInQueue)
		log.Err(err)
	}

	message.Content = "<@" + msg.Author.ID + ">, " + player.GetName() + " is level **" + strconv.Itoa(player.Level) + "**"
	return message, nil
}
