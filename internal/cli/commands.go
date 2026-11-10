package cli

import (
	"fmt"
	"strconv"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/discord"
)

func runAuth(args []string) error {
	if len(args) == 0 {
		return usagef("auth: missing subcommand (status)")
	}

	switch args[0] {
	case "status":
		return authStatus(args[1:])
	case "login":
		return authLogin(args[1:])
	default:
		return usagef("auth: unknown subcommand %q", args[0])
	}
}

func authStatus(args []string) error {
	fs := newFlagSet("auth status")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if err := noArgs(fs, positional); err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	me, err := client.Me()
	if err != nil {
		return err
	}

	source := "keyring"
	if authFromEnv() {
		source = "env"
	}

	return writeOKWith("account", newUser(*me), -1, map[string]any{
		"token_source": source,
	})
}

func runGuild(args []string) error {
	if len(args) == 0 {
		return usagef("guild: missing subcommand (list, get)")
	}

	switch args[0] {
	case "list":
		return guildList(args[1:])
	case "get":
		return guildGet(args[1:])
	default:
		return usagef("guild: unknown subcommand %q", args[0])
	}
}

func guildList(args []string) error {
	fs := newFlagSet("guild list")
	limit := fs.Uint("limit", 0, "maximum number of guilds; 0 means all")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if err := noArgs(fs, positional); err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	var guilds []discord.Guild
	if *limit == 0 {
		guilds, err = client.AllGuilds()
	} else {
		guilds, err = client.Guilds(*limit)
	}
	if err != nil {
		return err
	}

	out := make([]guildJSON, 0, len(guilds))
	for _, guild := range guilds {
		out = append(out, newGuild(guild))
	}

	return writeOK("guilds", out, len(out))
}

func guildGet(args []string) error {
	fs := newFlagSet("guild get")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}

	raw, err := requireArgs(fs, positional, "guild-id")
	if err != nil {
		return err
	}

	id, err := parseSnowflake(raw, "guild-id")
	if err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	guild, err := client.GuildWithCount(discord.GuildID(id))
	if err != nil {
		return err
	}

	return writeOK("guild", newGuildWithCount(*guild), -1)
}

func runChannel(args []string) error {
	if len(args) == 0 {
		return usagef("channel: missing subcommand (list, get)")
	}

	switch args[0] {
	case "list":
		return channelList(args[1:])
	case "get":
		return channelGet(args[1:])
	default:
		return usagef("channel: unknown subcommand %q", args[0])
	}
}

func channelList(args []string) error {
	fs := newFlagSet("channel list")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}

	raw, err := requireArgs(fs, positional, "guild-id")
	if err != nil {
		return err
	}

	id, err := parseSnowflake(raw, "guild-id")
	if err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	channels, err := client.Channels(discord.GuildID(id))
	if err != nil {
		return err
	}

	out := make([]channelJSON, 0, len(channels))
	for _, channel := range channels {
		out = append(out, newChannel(channel))
	}

	return writeOK("channels", out, len(out))
}

func channelGet(args []string) error {
	fs := newFlagSet("channel get")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}

	raw, err := requireArgs(fs, positional, "channel-id")
	if err != nil {
		return err
	}

	id, err := parseSnowflake(raw, "channel-id")
	if err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	channel, err := client.Channel(discord.ChannelID(id))
	if err != nil {
		return err
	}

	return writeOK("channel", newChannel(*channel), -1)
}

func runMsg(args []string) error {
	if len(args) == 0 {
		return usagef("msg: missing subcommand (list, get)")
	}

	switch args[0] {
	case "list":
		return msgList(args[1:])
	case "get":
		return msgGet(args[1:])
	default:
		return usagef("msg: unknown subcommand %q", args[0])
	}
}

func msgList(args []string) error {
	fs := newFlagSet("msg list")
	limit := fs.Uint("limit", 50, "maximum number of messages (1-100)")
	before := fs.String("before", "", "only messages before this message id")
	after := fs.String("after", "", "only messages after this message id")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}

	raw, err := requireArgs(fs, positional, "channel-id")
	if err != nil {
		return err
	}

	channelID, err := parseSnowflake(raw, "channel-id")
	if err != nil {
		return err
	}

	if *before != "" && *after != "" {
		return usagef("msg list: -before and -after are mutually exclusive")
	}
	if *limit == 0 || *limit > 100 {
		return usagef("msg list: -limit must be between 1 and 100")
	}

	var beforeID, afterID discord.MessageID
	if *before != "" {
		id, err := parseSnowflake(*before, "before")
		if err != nil {
			return err
		}
		beforeID = discord.MessageID(id)
	}
	if *after != "" {
		id, err := parseSnowflake(*after, "after")
		if err != nil {
			return err
		}
		afterID = discord.MessageID(id)
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	var messages []discord.Message
	switch {
	case beforeID.IsValid():
		messages, err = client.MessagesBefore(discord.ChannelID(channelID), beforeID, *limit)
	case afterID.IsValid():
		messages, err = client.MessagesAfter(discord.ChannelID(channelID), afterID, *limit)
	default:
		messages, err = client.Messages(discord.ChannelID(channelID), *limit)
	}
	if err != nil {
		return err
	}

	out := make([]messageJSON, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		out = append(out, newMessage(messages[i]))
	}

	extra := map[string]any{}
	if uint(len(out)) == *limit && len(out) > 0 {
		extra["next_before"] = out[0].ID
	}

	return writeOKWith("messages", out, len(out), extra)
}

func msgGet(args []string) error {
	fs := newFlagSet("msg get")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(positional) != 2 {
		return usagef("msg get: expected <channel-id> <message-id>")
	}

	channelID, err := parseSnowflake(positional[0], "channel-id")
	if err != nil {
		return err
	}

	messageID, err := parseSnowflake(positional[1], "message-id")
	if err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	// GET /channels/{id}/messages/{id} rejects user tokens, so fetch a small
	// window around the id through the list endpoint instead.
	messages, err := client.MessagesAround(discord.ChannelID(channelID), discord.MessageID(messageID), 5)
	if err != nil {
		return err
	}

	for _, message := range messages {
		if uint64(message.ID) == messageID {
			return writeOK("message", newMessage(message), -1)
		}
	}

	return &exitError{code: exitHTTP, err: fmt.Errorf("message %d not found in channel %d", messageID, channelID)}
}

func runSearch(args []string) error {
	fs := newFlagSet("search")
	query := fs.String("query", "", "text to search for")
	channel := fs.String("channel", "", "only search this channel id")
	author := fs.String("author", "", "only search messages by this user id")
	offset := fs.Uint("offset", 0, "result offset")
	sortOrder := fs.String("sort", "desc", "sort order (asc, desc)")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}

	raw, err := requireArgs(fs, positional, "guild-id")
	if err != nil {
		return err
	}

	guildID, err := parseSnowflake(raw, "guild-id")
	if err != nil {
		return err
	}

	if *query == "" && *channel == "" && *author == "" {
		return usagef("search: at least one of -query, -channel or -author is required")
	}
	if *sortOrder != "asc" && *sortOrder != "desc" {
		return usagef("search: -sort must be asc or desc")
	}

	data := api.SearchData{
		Content:   *query,
		SortOrder: *sortOrder,
		Offset:    *offset,
	}

	if *channel != "" {
		id, err := parseSnowflake(*channel, "channel")
		if err != nil {
			return err
		}
		data.ChannelID = discord.ChannelID(id)
	}
	if *author != "" {
		id, err := parseSnowflake(*author, "author")
		if err != nil {
			return err
		}
		data.AuthorID = discord.UserID(id)
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	resp, err := client.Search(discord.GuildID(guildID), data)
	if err != nil {
		return err
	}

	var out []messageJSON
	for _, group := range resp.Messages {
		for i := len(group) - 1; i >= 0; i-- {
			out = append(out, newMessage(group[i]))
		}
	}

	return writeOKWith("messages", out, len(out), map[string]any{
		"total_results": resp.TotalResults,
		"offset":        *offset,
	})
}

func runThread(args []string) error {
	if len(args) == 0 {
		return usagef("thread: missing subcommand (list, archived)")
	}

	switch args[0] {
	case "list":
		return threadList(args[1:])
	case "archived":
		return threadArchived(args[1:])
	default:
		return usagef("thread: unknown subcommand %q", args[0])
	}
}

// threadList searches the threads of one channel. Listing the active threads of
// a whole guild is a bot-only endpoint, so a channel is required.
func threadList(args []string) error {
	fs := newFlagSet("thread list")
	query := fs.String("query", "", "filter by thread name")
	archived := fs.Bool("archived", false, "include archived threads")
	limit := fs.Uint("limit", 25, "maximum number of threads (1-100)")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}

	raw, err := requireArgs(fs, positional, "channel-id")
	if err != nil {
		return err
	}

	channelID, err := parseSnowflake(raw, "channel-id")
	if err != nil {
		return err
	}

	if *limit == 0 || *limit > 100 {
		return usagef("thread list: -limit must be between 1 and 100")
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	threads, err := client.SearchThreads(discord.ChannelID(channelID), api.SearchThreadsData{
		Name:     *query,
		Archived: *archived,
		Limit:    int(*limit),
		SortBy:   api.LastMessageTime,
	})
	if err != nil {
		return err
	}

	out := make([]channelJSON, 0, len(threads.Threads))
	for _, thread := range threads.Threads {
		out = append(out, newChannel(thread))
	}

	return writeOKWith("threads", out, len(out), map[string]any{
		"has_more":      threads.HasMore,
		"total_results": threads.TotalResults,
	})
}

func threadArchived(args []string) error {
	fs := newFlagSet("thread archived")
	limit := fs.Uint("limit", 50, "maximum number of threads (1-100)")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}

	raw, err := requireArgs(fs, positional, "channel-id")
	if err != nil {
		return err
	}

	channelID, err := parseSnowflake(raw, "channel-id")
	if err != nil {
		return err
	}

	if *limit == 0 || *limit > 100 {
		return usagef("thread archived: -limit must be between 1 and 100")
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	threads, err := client.PublicArchivedThreads(discord.ChannelID(channelID), discord.Timestamp{}, *limit)
	if err != nil {
		return err
	}

	out := make([]channelJSON, 0, len(threads.Threads))
	for _, thread := range threads.Threads {
		out = append(out, newChannel(thread))
	}

	return writeOKWith("threads", out, len(out), map[string]any{
		"has_more": threads.More,
	})
}

func runMember(args []string) error {
	if len(args) == 0 {
		return usagef("member: missing subcommand (get)")
	}

	switch args[0] {
	case "get":
		return memberGet(args[1:])
	default:
		return usagef("member: unknown subcommand %q", args[0])
	}
}

func memberGet(args []string) error {
	fs := newFlagSet("member get")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(positional) != 2 {
		return usagef("member get: expected <guild-id> <user-id>")
	}

	guildID, err := parseSnowflake(positional[0], "guild-id")
	if err != nil {
		return err
	}

	userID, err := parseSnowflake(positional[1], "user-id")
	if err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	member, err := client.Member(discord.GuildID(guildID), discord.UserID(userID))
	if err != nil {
		return err
	}

	return writeOK("member", newMember(*member), -1)
}

func runPin(args []string) error {
	if len(args) == 0 {
		return usagef("pin: missing subcommand (list)")
	}

	switch args[0] {
	case "list":
		return pinList(args[1:])
	default:
		return usagef("pin: unknown subcommand %q", args[0])
	}
}

func pinList(args []string) error {
	fs := newFlagSet("pin list")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}

	raw, err := requireArgs(fs, positional, "channel-id")
	if err != nil {
		return err
	}

	channelID, err := parseSnowflake(raw, "channel-id")
	if err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	messages, err := client.PinnedMessages(discord.ChannelID(channelID))
	if err != nil {
		return err
	}

	out := make([]messageJSON, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		out = append(out, newMessage(messages[i]))
	}

	return writeOK("messages", out, len(out))
}

func runDM(args []string) error {
	if len(args) == 0 {
		return usagef("dm: missing subcommand (list)")
	}

	switch args[0] {
	case "list":
		return dmList(args[1:])
	default:
		return usagef("dm: unknown subcommand %q", args[0])
	}
}

func dmList(args []string) error {
	fs := newFlagSet("dm list")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if err := noArgs(fs, positional); err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	channels, err := client.PrivateChannels()
	if err != nil {
		return err
	}

	out := make([]channelJSON, 0, len(channels))
	for _, channel := range channels {
		out = append(out, newChannel(channel))
	}

	return writeOK("channels", out, len(out))
}

func runUser(args []string) error {
	if len(args) == 0 {
		return usagef("user: missing subcommand (me, get)")
	}

	switch args[0] {
	case "me":
		return userMe(args[1:])
	case "get":
		return userGet(args[1:])
	default:
		return usagef("user: unknown subcommand %q", args[0])
	}
}

func userMe(args []string) error {
	fs := newFlagSet("user me")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if err := noArgs(fs, positional); err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	me, err := client.Me()
	if err != nil {
		return err
	}

	return writeOK("user", newUser(*me), -1)
}

func userGet(args []string) error {
	fs := newFlagSet("user get")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}

	raw, err := requireArgs(fs, positional, "user-id")
	if err != nil {
		return err
	}

	userID, err := parseSnowflake(raw, "user-id")
	if err != nil {
		return err
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	user, err := client.User(discord.UserID(userID))
	if err != nil {
		return err
	}

	return writeOK("user", newUser(*user), -1)
}

func parseSnowflake(raw, name string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, usagef("invalid %s %q: expected a numeric snowflake", name, raw)
	}

	return id, nil
}
