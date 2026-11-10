package cli

import (
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
)

type userJSON struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name,omitempty"`
	Bot        bool   `json:"bot,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
}

func newUser(u discord.User) userJSON {
	return userJSON{
		ID:         u.ID.String(),
		Username:   u.Username,
		GlobalName: u.DisplayName,
		Bot:        u.Bot,
		AvatarURL:  u.AvatarURL(),
	}
}

type guildJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Owner       bool   `json:"owner,omitempty"`
	MemberCount uint64 `json:"member_count,omitempty"`
	OnlineCount uint64 `json:"online_count,omitempty"`
	Description string `json:"description,omitempty"`
	IconURL     string `json:"icon_url,omitempty"`
}

func newGuild(g discord.Guild) guildJSON {
	out := guildJSON{
		ID:          g.ID.String(),
		Name:        g.Name,
		Owner:       g.Owner,
		Description: g.Description,
	}
	if g.Icon != "" {
		out.IconURL = g.IconURL()
	}
	return out
}

func newGuildWithCount(g discord.Guild) guildJSON {
	out := newGuild(g)
	out.MemberCount = g.ApproximateMembers
	out.OnlineCount = g.ApproximatePresences
	return out
}

type channelJSON struct {
	ID            string     `json:"id"`
	GuildID       string     `json:"guild_id,omitempty"`
	ParentID      string     `json:"parent_id,omitempty"`
	Type          int        `json:"type"`
	TypeName      string     `json:"type_name"`
	Name          string     `json:"name,omitempty"`
	Topic         string     `json:"topic,omitempty"`
	NSFW          bool       `json:"nsfw,omitempty"`
	Position      int        `json:"position,omitempty"`
	LastMessageID string     `json:"last_message_id,omitempty"`
	Recipients    []userJSON `json:"recipients,omitempty"`
}

func newChannel(c discord.Channel) channelJSON {
	out := channelJSON{
		ID:            c.ID.String(),
		Type:          int(c.Type),
		TypeName:      channelTypeName(c.Type),
		Name:          c.Name,
		Topic:         c.Topic,
		NSFW:          c.NSFW,
		Position:      c.Position,
		LastMessageID: c.LastMessageID.String(),
	}
	if c.GuildID.IsValid() {
		out.GuildID = c.GuildID.String()
	}
	if c.ParentID.IsValid() {
		out.ParentID = c.ParentID.String()
	}
	for _, r := range c.DMRecipients {
		out.Recipients = append(out.Recipients, newUser(r))
	}
	return out
}

func channelTypeName(t discord.ChannelType) string {
	switch t {
	case discord.GuildText:
		return "guild_text"
	case discord.DirectMessage:
		return "dm"
	case discord.GuildVoice:
		return "guild_voice"
	case discord.GroupDM:
		return "group_dm"
	case discord.GuildCategory:
		return "guild_category"
	case discord.GuildAnnouncement:
		return "guild_announcement"
	case discord.GuildAnnouncementThread, discord.GuildPublicThread:
		return "public_thread"
	case discord.GuildPrivateThread:
		return "private_thread"
	case discord.GuildStageVoice:
		return "guild_stage_voice"
	case discord.GuildForum:
		return "guild_forum"
	default:
		return "unknown"
	}
}

type attachmentJSON struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
	Size        uint64 `json:"size"`
	URL         string `json:"url"`
	Width       uint   `json:"width,omitempty"`
	Height      uint   `json:"height,omitempty"`
}

type reactionJSON struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Me    bool   `json:"me,omitempty"`
}

type referenceJSON struct {
	MessageID string `json:"message_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
	GuildID   string `json:"guild_id,omitempty"`
}

type messageJSON struct {
	ID              string           `json:"id"`
	ChannelID       string           `json:"channel_id"`
	GuildID         string           `json:"guild_id,omitempty"`
	Author          userJSON         `json:"author"`
	Timestamp       string           `json:"timestamp"`
	EditedTimestamp string           `json:"edited_timestamp,omitempty"`
	Content         string           `json:"content"`
	Pinned          bool             `json:"pinned,omitempty"`
	MentionEveryone bool             `json:"mention_everyone,omitempty"`
	Attachments     []attachmentJSON `json:"attachments,omitempty"`
	EmbedCount      int              `json:"embed_count,omitempty"`
	StickerCount    int              `json:"sticker_count,omitempty"`
	Reactions       []reactionJSON   `json:"reactions,omitempty"`
	Reference       *referenceJSON   `json:"reference,omitempty"`
	ThreadID        string           `json:"thread_id,omitempty"`
	URL             string           `json:"url"`
}

func newMessage(m discord.Message) messageJSON {
	out := messageJSON{
		ID:              m.ID.String(),
		ChannelID:       m.ChannelID.String(),
		Author:          newUser(m.Author),
		Timestamp:       formatTimestamp(m.Timestamp),
		Content:         m.Content,
		Pinned:          m.Pinned,
		MentionEveryone: m.MentionEveryone,
		EmbedCount:      len(m.Embeds),
		StickerCount:    len(m.Stickers),
		URL:             m.URL(),
	}

	if m.GuildID.IsValid() {
		out.GuildID = m.GuildID.String()
	}
	if m.EditedTimestamp.IsValid() {
		out.EditedTimestamp = formatTimestamp(m.EditedTimestamp)
	}
	for _, a := range m.Attachments {
		out.Attachments = append(out.Attachments, attachmentJSON{
			ID:          a.ID.String(),
			Filename:    a.Filename,
			ContentType: a.ContentType,
			Size:        a.Size,
			URL:         string(a.URL),
			Width:       a.Width,
			Height:      a.Height,
		})
	}
	for _, r := range m.Reactions {
		out.Reactions = append(out.Reactions, reactionJSON{
			Emoji: r.Emoji.Name,
			Count: r.Count,
			Me:    r.Me,
		})
	}
	if m.Reference != nil {
		ref := &referenceJSON{
			MessageID: m.Reference.MessageID.String(),
			ChannelID: m.Reference.ChannelID.String(),
			GuildID:   m.Reference.GuildID.String(),
		}
		out.Reference = ref
	}
	if m.Thread != nil && m.Thread.ID.IsValid() {
		out.ThreadID = m.Thread.ID.String()
	}

	return out
}

type memberJSON struct {
	User     userJSON `json:"user"`
	Nick     string   `json:"nick,omitempty"`
	JoinedAt string   `json:"joined_at,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

func newMember(m discord.Member) memberJSON {
	out := memberJSON{
		User: newUser(m.User),
		Nick: m.Nick,
	}
	if m.Joined.IsValid() {
		out.JoinedAt = formatTimestamp(m.Joined)
	}
	for _, role := range m.RoleIDs {
		out.Roles = append(out.Roles, role.String())
	}
	return out
}

func formatTimestamp(t discord.Timestamp) string {
	if !t.IsValid() {
		return ""
	}
	return time.Time(t).Format(time.RFC3339)
}
