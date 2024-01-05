package botmanager

import (
	"fmt"
	"gamemediabot/lib"
	"gamemediabot/xapi"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	globalManger *BotManager = nil
)

type CommandArg struct {
	s           *discordgo.Session
	m           *discordgo.MessageCreate
	token       []string
	originalMsg string
	commandName string
}

func NewCommandArg(s *discordgo.Session, m *discordgo.MessageCreate, token []string, originalMsg, commandName string) *CommandArg {
	return &CommandArg{
		s:           s,
		m:           m,
		token:       token,
		originalMsg: originalMsg,
		commandName: commandName,
	}
}

type Command struct {
	Name    string
	handler func(arg *CommandArg, manager *BotManager)
	summary string
	detail  string
}

type BatchTweet struct {
	Tweet   *xapi.XTweet
	Medias  []xapi.XMedia
	GetTime time.Time
}

var (
	NULL_FILTER_FILED_INT = -1
)

type Fileter struct {
	Name             string
	IsIncludeRetweet bool
	MinLikeCount     int
	MaxLikeCount     int
	MinRetweetCount  int
	MaxRetweetCount  int
	IncluedUserIDs   map[string]bool
	ExcluedUserIDs   map[string]bool
	AndKeywords      []string
	OrKeywords       []string
	MAX_TWEET_COUNT  int
}

func NewFilter(name string) *Fileter {
	return &Fileter{
		Name:             name,
		IsIncludeRetweet: false,
		MinLikeCount:     NULL_FILTER_FILED_INT,
		MaxLikeCount:     NULL_FILTER_FILED_INT,
		MinRetweetCount:  NULL_FILTER_FILED_INT,
		MaxRetweetCount:  NULL_FILTER_FILED_INT,
		IncluedUserIDs:   make(map[string]bool),
		ExcluedUserIDs:   make(map[string]bool),
		AndKeywords:      make([]string, 0),
		OrKeywords:       make([]string, 0),
		MAX_TWEET_COUNT:  NULL_FILTER_FILED_INT,
	}
}

type BotManager struct {
	BotUserInfo        *discordgo.User
	allowedChannels    map[string]bool
	targetUsers        map[string]xapi.XUser
	commands           map[string]*Command
	filters            map[string]*Fileter
	client             *xapi.XClinet
	discordSession     *discordgo.Session
	batchDurationMinu  int
	lastBatchDate      time.Time
	nextBatchDate      time.Time
	lastUpdateCashDate time.Time
	isRunningBatch     bool
	BatchTweetsCash    []BatchTweet
	BatchTweetIDs      []string
	MAX_RESTORE_TWEETS int
	TweetCashMin       int
}

func SetGlobalManager(manager *BotManager) {
	globalManger = manager
}

func GetGlobalManager() *BotManager {
	return globalManger
}

func NewBotManager(discordgoSession *discordgo.Session, client *xapi.XClinet, batchDurationMinu int) *BotManager {

	manager := &BotManager{
		allowedChannels:    make(map[string]bool),
		targetUsers:        make(map[string]xapi.XUser),
		filters:            make(map[string]*Fileter),
		client:             client,
		discordSession:     discordgoSession,
		batchDurationMinu:  batchDurationMinu,
		lastBatchDate:      time.Now().UTC(),
		lastUpdateCashDate: time.Now().UTC(),
		isRunningBatch:     false,
		BatchTweetsCash:    make([]BatchTweet, 0),
		MAX_RESTORE_TWEETS: 85,
		TweetCashMin:       10,
	}

	manager.nextBatchDate = manager.lastBatchDate.Add(time.Duration(batchDurationMinu) * time.Minute)

	fmt.Println("Initial lastBatchDate : ", manager.lastBatchDate.Format("2006-01-02 15:04:05"))
	fmt.Println("Initial nextBatchDate : ", manager.nextBatchDate.Format("2006-01-02 15:04:05"))

	initialTargetUsers := []string{
		"AUTOMATONJapan",
		"Indie_FreaksJP",
		"gamespark",
		"denfaminicogame",
	}

	for _, userName := range initialTargetUsers {
		user, err := manager.client.GetUser(userName)
		if err != nil {
			continue
		}
		manager.AddTargetUser(user)
	}

	manager.setCommands()
	manager.discordSession.AddHandler(onDiscordMessageCreate)
	return manager
}

func (manager *BotManager) SendNormalMessage(channelId string, title string, msg string, fileds []*discordgo.MessageEmbedField) {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: msg,
		Color:       0x00ff00,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: manager.BotUserInfo.AvatarURL("20"),
		},
	}

	if fileds != nil {
		embed.Fields = fileds
	}
	_, err := manager.discordSession.ChannelMessageSendEmbed(channelId, embed)
	if err != nil {
		log.Println("Error sending normal embed message\n" + err.Error())
	}
}

func (manager *BotManager) SendErrorMessage(channelId string, title string, msg string, fileds []*discordgo.MessageEmbedField) {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: msg,
		Color:       0xff0000,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: manager.BotUserInfo.AvatarURL("20"),
		},
	}

	if fileds != nil {
		embed.Fields = fileds
	}
	_, err := manager.discordSession.ChannelMessageSendEmbed(channelId, embed)
	if err != nil {
		log.Println("Error sending error embed message\n" + err.Error())
	}
}

func (manager *BotManager) setCommands() {
	manager.commands = make(map[string]*Command)
	commands := make([]*Command, 0)
	commands = append(commands, &Command{
		Name:    "help",
		handler: onHelpCommand,
		summary: "コマンド一覧や詳細を表示します",
		detail:  "【コマンド】 " + "\n\t\t**help\t(コマンド名)**\n" + "【機能】\n" + "\t・コマンドの一覧を表示します\n" + "\t・コマンド名を指定すると詳細を表示します\n",
	})
	commands = append(commands, &Command{
		Name:    "allow",
		handler: onAllowChannelCommand,
		summary: "このチャンネルでの発言を許可します",
	})
	commands = append(commands, &Command{
		Name:    "settings",
		handler: onSettingsCommand,
		summary: "設定値を表示します",
	})
	commands = append(commands, &Command{
		Name:    "targets",
		handler: onTargetsCommand,
		summary: "登録済みのユーザー一覧を表示します",
	})
	commands = append(commands, &Command{
		Name:    "addtarget",
		handler: onAddTargetUserCommand,
		summary: "ユーザーを追加します",
	})
	commands = append(commands, &Command{
		Name:    "change_duration",
		handler: onChangeBatchDurationCommand,
		summary: "バッチ処理の間隔を変更します",
	})

	commands = append(commands, &Command{
		Name:    "test",
		handler: onTestCommand,
		summary: "開発用コマンドです",
	})
	commands = append(commands, &Command{
		Name:    "removetarget",
		handler: onRemoveTargetUserCommand,
		summary: "ユーザーを削除します",
	})
	commands = append(commands, &Command{
		Name:    "setfilter",
		handler: onSetFileterCommand,
		summary: "フィルターを設定します",
	})
	commands = append(commands, &Command{
		Name:    "applyfilter",
		handler: onFilterGetCommand,
		summary: "フィルターを適用してツイートを表示します",
	})
	commands = append(commands, &Command{
		Name:    "filters",
		handler: onFiltersCommand,
		summary: "フィルター一覧や詳細を表示します",
	})
	commands = append(commands, &Command{
		Name:    "removefilter",
		handler: onRemoveFilterCommand,
		summary: "フィルターを削除します",
	})

	for _, command := range commands {
		manager.commands[command.Name] = command
	}
}

func (manager *BotManager) setBatchDurationMinu(minute int) {
	manager.batchDurationMinu = minute
}

func (manager *BotManager) GetNextBatchTime() time.Time {
	return manager.nextBatchDate
}

func (manager *BotManager) AddAllowedChannel(channelId string) {
	manager.allowedChannels[channelId] = true
}

func (manager *BotManager) AddTargetUser(user xapi.XUser) {
	manager.targetUsers[user.Id] = user
}

func (manager *BotManager) AddBatchTweet(tweetId string) {
	if len(manager.BatchTweetIDs) >= manager.MAX_RESTORE_TWEETS {
		manager.BatchTweetIDs = manager.BatchTweetIDs[1:]
	}
	manager.BatchTweetIDs = append(manager.BatchTweetIDs, tweetId)
}

func (manager *BotManager) refleshBatchCash() {
	results, err := manager.client.GetTweetsByID(manager.BatchTweetIDs)
	if err != nil {
		return
	}

	now := time.Now().UTC()
	author := xapi.XUser{}
	btweets := ConvertBtweet(&results, author, now)

	result := make([]BatchTweet, 0)
	for idx := range btweets {
		authorId := btweets[idx].Tweet.AuthorID
		author, exist := manager.targetUsers[authorId]
		//ユーザーが削除されている場合は無視
		if !exist {
			fmt.Println("User is not exist : ", authorId)
			continue
		}
		btweets[idx].Tweet.Author = author
		result = append(result, btweets[idx])
	}
	manager.BatchTweetsCash = result
	manager.lastUpdateCashDate = now
}

func (manager *BotManager) getTweets() map[string]xapi.XTweetsResult {
	maxResults := 5
	results := make(map[string]xapi.XTweetsResult)
	//同時刻のツイートを取得しないように、最後に取得した時間の2秒後から取得する
	startTime := manager.lastBatchDate.Add(2 * time.Second)

	for _, user := range manager.targetUsers {
		result, err := manager.client.GetTweetsByUserId(&user, maxResults, &startTime)
		if err != nil {
			continue
		}
		results[user.Id] = result
	}

	return results
}

func (manager *BotManager) sentBatchMessages(channelId string, tweets map[string]xapi.XTweetsResult) {

	now := time.Now().UTC()

	for userId, result := range tweets {
		author := manager.targetUsers[userId]
		btweets := ConvertBtweet(&result, author, now)

		for idx := range btweets {
			embed := makeNotionMessage(nil, manager, &btweets[idx])
			manager.discordSession.ChannelMessageSendEmbed(channelId, embed)
		}
	}
}

func (manager *BotManager) batchLoop() {
	for {
		dulation := time.Duration(manager.batchDurationMinu) * time.Minute
		time.Sleep(dulation)
		log.Println("Batch Executed : ", time.Now().Format("2006-01-02 15:04:05"))

		tweets := manager.getTweets()
		//最終探索時間の更新
		manager.lastBatchDate = time.Now().UTC()
		manager.nextBatchDate = manager.lastBatchDate.Add(dulation)

		for channelId := range manager.allowedChannels {
			manager.sentBatchMessages(channelId, tweets)
			for _, result := range tweets {
				for idx, _ := range result.Data {
					manager.AddBatchTweet(result.Data[idx].Id)
				}
			}
		}
	}
}

func (manager *BotManager) Start() {
	if manager.isRunningBatch {
		return
	}
	manager.isRunningBatch = true
	manager.BotUserInfo = manager.discordSession.State.User
	go manager.batchLoop()
}

func onDiscordMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	commandTriger := "!mediabot"
	msg := m.Content

	tokens1 := strings.Split(msg, "\n")
	tokens := make([]string, 0)

	tokens = append(tokens, strings.Split(tokens1[0], " ")...)
	for _, token := range tokens1[1:] {
		tokens = append(tokens, token)
	}

	if commandTriger != tokens[0] {
		return
	}

	manager := GetGlobalManager()

	if len(tokens) < 2 {
		onInvalidCommand(s, m, manager)
		return
	}

	commandName := tokens[1]

	commandArg := NewCommandArg(s, m, tokens, msg, commandName)

	//コマンドの実行
	if command, ok := manager.commands[commandName]; ok {
		log.Printf("Execute command: %s", commandName)
		command.handler(commandArg, manager)
	} else {
		fmt.Println("Invalid command: ", commandName)
		onInvalidCommand(s, m, manager)
	}
}

func sendMessage(s *discordgo.Session, channelID string, msg string) {
	_, err := s.ChannelMessageSend(channelID, msg)
	if err != nil {
		errmsg := fmt.Sprintf("Error sending message : cnannelId[%s], msg[%s]", channelID, msg)
		log.Println(errmsg)
	}
}

func onAllowChannelCommand(arg *CommandArg, manager *BotManager) {

	channelId := arg.m.ChannelID
	manager.AddAllowedChannel(channelId)

	msg := "このチャンネルでの発言を許可しました"
	manager.SendNormalMessage(channelId, "", msg, nil)
}

func onChangeBatchDurationCommand(arg *CommandArg, manager *BotManager) {
	maxDuration := 120
	minDuariont := 1

	if len(arg.token) < 3 {
		errmsg := "分を指定してください"
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	minu, err := strconv.Atoi(arg.token[2])
	if err != nil {
		errmsg := "分には数値を指定してください(Max : 120, Min : 1)"
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	if minu >= maxDuration || minu < minDuariont {
		errmsg := fmt.Sprintf("分の値は%d以上%d以下にしてください", minDuariont, maxDuration)
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	oldDuarion := manager.batchDurationMinu
	manager.setBatchDurationMinu(minu)
	msg := fmt.Sprintf("バッチ処理の間隔を%d(分)から%d(分)に変更しました\n次回のバッチ処理後に適用されます", oldDuarion, minu)
	manager.SendNormalMessage(arg.m.ChannelID, "", msg, nil)
}

func onAddTargetUserCommand(arg *CommandArg, manager *BotManager) {
	if len(arg.token) < 3 {
		errmsg := "ユーザー名を指定してください"
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	userName := arg.token[2]
	user, err := manager.client.GetUser(userName)

	if err != nil {
		errmsg := fmt.Sprintf("ユーザー名[%s]のユーザーが見つかりませんでした", userName)
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	manager.AddTargetUser(user)
	msg := fmt.Sprintf("ユーザー名[%s]のユーザーを追加しました", user.Username)
	manager.SendNormalMessage(arg.m.ChannelID, "", msg, nil)

	for id, target := range manager.targetUsers {
		fmt.Printf("key = %s : user = %+v\n", id, target)
	}
}

func onRemoveTargetUserCommand(arg *CommandArg, manager *BotManager) {
	if len(arg.token) < 3 {
		errmsg := "ユーザー名を指定してください"
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	userName := arg.token[2]
	user, err := manager.client.GetUser(userName)

	if err != nil {
		errmsg := fmt.Sprintf("ユーザー名[%s]のユーザーが見つかりませんでした", userName)
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	delete(manager.targetUsers, user.Id)
	msg := fmt.Sprintf("ユーザー名[%s]のユーザーを削除しました", user.Username)
	manager.SendNormalMessage(arg.m.ChannelID, "", msg, nil)
}

func onTargetsCommand(arg *CommandArg, manager *BotManager) {
	title := "登録済みのユーザー一覧\n"
	msg := ""
	for _, user := range manager.targetUsers {
		link := fmt.Sprintf("https://twitter.com/%s", user.Username)
		msg += fmt.Sprintf("-\t%s  ([Link](%s))\n", user.Username, link)
	}
	manager.SendNormalMessage(arg.m.ChannelID, title, msg, nil)
}

func onSettingsCommand(arg *CommandArg, manager *BotManager) {

	nextLocalTimeStr, err := lib.UTCtimeToLoaclTime(manager.GetNextBatchTime())
	timeZone := "Asia/Tokyo"

	if err != nil {
		DifTime := time.Hour * 9
		nextLocal := manager.GetNextBatchTime().Add(DifTime)
		nextLocalTimeStr = nextLocal.Format("2006-01-02 15:04:05")
		timeZone = "Japan"
	}

	fileds := make([]*discordgo.MessageEmbedField, 0)
	fileds = append(fileds, &discordgo.MessageEmbedField{
		Name:   "バッチ処理間隔(分)",
		Value:  fmt.Sprintf("%d", manager.batchDurationMinu),
		Inline: true,
	})
	fileds = append(fileds, &discordgo.MessageEmbedField{
		Name:   "次回のバッチ処理",
		Value:  fmt.Sprintf("%s (%s)", nextLocalTimeStr, timeZone),
		Inline: true,
	})
	fileds = append(fileds, &discordgo.MessageEmbedField{
		Name:   "保存ツイート数",
		Value:  fmt.Sprintf("%d", len(manager.BatchTweetIDs)),
		Inline: false,
	})

	manager.SendNormalMessage(arg.m.ChannelID, "設定値一覧\n", "", fileds)
}

func onHelpCommand(arg *CommandArg, manager *BotManager) {
	if len(arg.token) < 3 {
		keys := make([]string, 0, len(manager.commands))
		for key := range manager.commands {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		msg := "コマンド一覧\n"
		for _, name := range keys {
			command := manager.commands[name]
			msg += fmt.Sprintf("**%s**\n\tー\t%s\n", command.Name, command.summary)
		}
		msg += "各コマンドの詳細は「!mediabot help <コマンド名>」で確認できます\n"

		sendMessage(arg.s, arg.m.ChannelID, msg)
		return
	}

	commandName := arg.token[2]
	command, exist := manager.commands[commandName]
	if !exist {
		errmsg := "指定されたコマンドは存在しません"
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	msg := command.detail
	sendMessage(arg.s, arg.m.ChannelID, msg)
}

func onSetFileterCommand(arg *CommandArg, manager *BotManager) {
	params := paramParse(arg.token)

	filterName, exist := params["name"]
	if !exist {
		errmsg := "フィルター名を指定してください"
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	filter := NewFilter(filterName)

	if minLikeCntStr, exist := params["min_like"]; exist {
		minLikeCnt, err := strconv.Atoi(minLikeCntStr)
		if err != nil {
			errmsg := "min_likeには数値を指定してください"
			title := arg.commandName
			manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
			return
		}
		filter.MinLikeCount = minLikeCnt
	}

	if maxLikeCntStr, exist := params["max_like"]; exist {
		maxLikeCnt, err := strconv.Atoi(maxLikeCntStr)
		if err != nil {
			errmsg := "max_likeには数値を指定してください"
			title := arg.commandName
			manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
			return
		}
		filter.MaxLikeCount = maxLikeCnt
	}

	if minRetweetCntStr, exist := params["min_retweet"]; exist {
		minRetweetCnt, err := strconv.Atoi(minRetweetCntStr)
		if err != nil {
			errmsg := "min_retweetには数値を指定してください"
			title := arg.commandName
			manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
			return
		}
		filter.MinRetweetCount = minRetweetCnt
	}

	if maxRetweetCntStr, exist := params["max_retweet"]; exist {
		maxRetweetCnt, err := strconv.Atoi(maxRetweetCntStr)
		if err != nil {
			errmsg := "max_retweetには数値を指定してください"
			title := arg.commandName
			manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
			return
		}
		filter.MaxRetweetCount = maxRetweetCnt
	}

	if incluedUserIDsStr, exist := params["inclued_users"]; exist {
		incluedUserIDs := strings.Split(incluedUserIDsStr, "&")
		for _, userID := range incluedUserIDs {
			filter.IncluedUserIDs[userID] = true
		}
	}

	if excluedUserIDsStr, exist := params["exclued_users"]; exist {
		excluedUserIDs := strings.Split(excluedUserIDsStr, "&")
		for _, userID := range excluedUserIDs {
			filter.ExcluedUserIDs[userID] = true
		}
	}

	if andKeywordsStr, exist := params["and_keywords"]; exist {
		andKeywords := strings.Split(andKeywordsStr, "&")
		for _, keyword := range andKeywords {
			filter.AndKeywords = append(filter.AndKeywords, keyword)
		}
	}

	if orKeywordsStr, exist := params["or_keywords"]; exist {
		orKeywords := strings.Split(orKeywordsStr, "&")
		for _, keyword := range orKeywords {
			filter.OrKeywords = append(filter.OrKeywords, keyword)
		}
	}

	if maxTweetCntStr, exist := params["max_tweets"]; exist {
		maxTweetCnt, err := strconv.Atoi(maxTweetCntStr)
		if err != nil {
			errmsg := "max_tweetsには数値を指定してください"
			title := arg.commandName
			manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
			return
		}
		filter.MAX_TWEET_COUNT = maxTweetCnt
	}

	manager.filters[filterName] = filter
	msg := fmt.Sprintf("フィルター[%s]を設定しました", filterName)
	manager.SendNormalMessage(arg.m.ChannelID, "", msg, nil)
}

func onFilterGetCommand(arg *CommandArg, manager *BotManager) {
	if len(arg.token) < 3 {
		errmsg := "フィルター名を指定してください"
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	filterName := arg.token[2]
	filter, exist := manager.filters[filterName]
	if !exist {
		errmsg := fmt.Sprintf("指定されたフィルター[%s]は存在しません", filterName)
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	now := time.Now().UTC()
	refleshTime := manager.lastUpdateCashDate.Add(time.Duration(manager.TweetCashMin) * time.Minute)

	if now.After(refleshTime) {
		manager.refleshBatchCash()
	}

	threadTitle := fmt.Sprintf("\"%s\"による結果", filterName)
	thread, err := arg.s.MessageThreadStart(arg.m.ChannelID, arg.m.ID, threadTitle, 0)
	if err != nil {
		fmt.Println(err)
	}

	sendCnt := 0
	for idx, _ := range manager.BatchTweetsCash {
		if IsMatchFilter(&manager.BatchTweetsCash[idx], filter) {
			embed := makeFilterNotionMessage(arg, manager, &manager.BatchTweetsCash[idx])
			arg.s.ChannelMessageSendEmbed(thread.ID, embed)
			sendCnt++
			if (filter.MAX_TWEET_COUNT != NULL_FILTER_FILED_INT) && (sendCnt >= filter.MAX_TWEET_COUNT) {
				break
			}
		}
	}

	fmt.Printf("thread : %+v\n", thread)
	isArchived := true
	_, err = arg.s.ChannelEditComplex(thread.ID, &discordgo.ChannelEdit{
		Archived: &isArchived,
	})

	msg := fmt.Sprintf("フィルター[%s]を適用しました\nスレッドを確認してください", filterName)
	manager.SendNormalMessage(arg.m.ChannelID, "", msg, nil)
}

func onFiltersCommand(arg *CommandArg, manager *BotManager) {

	if len(arg.token) < 3 {
		keys := make([]string, 0, len(manager.filters))
		for key := range manager.filters {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		msg := "フィルター一覧\n"
		for _, name := range keys {
			filter := manager.filters[name]
			msg += fmt.Sprintf("-\t%s\n", filter.Name)
		}
		manager.SendNormalMessage(arg.m.ChannelID, "", msg, nil)
		return
	}

	filterName := arg.token[2]
	filter, exist := manager.filters[filterName]
	if !exist {
		errmsg := fmt.Sprintf("指定されたフィルター[%s]は存在しません", filterName)
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	title := fmt.Sprintf("%sの詳細", filter.Name)

	getCntValue := func(value int) string {
		if value == NULL_FILTER_FILED_INT {
			return "ー"
		}
		return fmt.Sprintf("%d", value)
	}

	fields := make([]*discordgo.MessageEmbedField, 0)
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "MinLikeCount",
		Value:  fmt.Sprintf("%s", getCntValue(filter.MinLikeCount)),
		Inline: true,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "MaxLikeCount",
		Value:  fmt.Sprintf("%s", getCntValue(filter.MaxLikeCount)),
		Inline: true,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Inline: false,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "MinRTCount",
		Value:  fmt.Sprintf("%s", getCntValue(filter.MinRetweetCount)),
		Inline: true,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "MaxRTCount",
		Value:  fmt.Sprintf("%s", getCntValue(filter.MaxRetweetCount)),
		Inline: true,
	})

	getTargetsStr := func(targets map[string]bool) string {
		if len(targets) == 0 {
			return "ー"
		}
		result := ""
		for key := range targets {
			result += fmt.Sprintf("-\t\"%s\"\n", key)
		}
		return result
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "IncluedUserIDs",
		Value:  fmt.Sprintf("%s", getTargetsStr(filter.IncluedUserIDs)),
		Inline: false,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "ExcluedUserIDs",
		Value:  fmt.Sprintf("%s", getTargetsStr(filter.ExcluedUserIDs)),
		Inline: false,
	})

	getKeywordsStr := func(keywords []string) string {
		if len(keywords) == 0 {
			return "ー"
		}
		result := ""
		for _, keyword := range keywords {
			result += fmt.Sprintf("\"%s\"\t", keyword)
		}
		return result
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "AndKeywords",
		Value:  fmt.Sprintf("%s", getKeywordsStr(filter.AndKeywords)),
		Inline: false,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "OrKeywords",
		Value:  fmt.Sprintf("%s", getKeywordsStr(filter.OrKeywords)),
		Inline: false,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "MAX_TWEET_COUNT",
		Value:  fmt.Sprintf("%s", getCntValue(filter.MAX_TWEET_COUNT)),
		Inline: false,
	})

	manager.SendNormalMessage(arg.m.ChannelID, title, "", fields)
}

func onRemoveFilterCommand(arg *CommandArg, manager *BotManager) {
	if len(arg.token) < 3 {
		errmsg := "フィルター名を指定してください"
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	filterName := arg.token[2]
	_, exist := manager.filters[filterName]
	if !exist {
		errmsg := fmt.Sprintf("指定されたフィルター[%s]は存在しません", filterName)
		title := arg.commandName
		manager.SendErrorMessage(arg.m.ChannelID, title, errmsg, nil)
		return
	}

	delete(manager.filters, filterName)
	msg := fmt.Sprintf("フィルター[%s]を削除しました", filterName)
	manager.SendNormalMessage(arg.m.ChannelID, "", msg, nil)
}

func onInvalidCommand(s *discordgo.Session, m *discordgo.MessageCreate, manager *BotManager) {
	msg := "不正なコマンドです。コマンド一覧は「!mediabot help」で確認できます。"
	sendMessage(s, m.ChannelID, msg)
}

func onTestCommand(arg *CommandArg, manager *BotManager) {
}

func paramParse(tokens []string) map[string]string {
	params := make(map[string]string)
	for _, token := range tokens {
		if strings.Contains(token, "=") {
			param := strings.Split(token, "=")
			params[param[0]] = param[1]
		}
	}
	return params
}

func makeNotionMessage(arg *CommandArg, manager *BotManager, btweet *BatchTweet) *discordgo.MessageEmbed {
	link := fmt.Sprintf("https://twitter.com/%s/status/%s", btweet.Tweet.Author.Username, btweet.Tweet.Id)
	userLink := fmt.Sprintf("https://twitter.com/%s", btweet.Tweet.Author.Username)
	title := fmt.Sprintf("@%s\tPosted！", btweet.Tweet.Author.Username)

	content := ""
	content += fmt.Sprintf("[Tweet](%s)\n", link)
	content += "────────────────────\n"

	content += "### " + btweet.Tweet.Text

	fileds := make([]*discordgo.MessageEmbedField, 0)
	fileds = append(fileds, &discordgo.MessageEmbedField{
		Name:   "ReTweetCount",
		Value:  fmt.Sprintf("%d", btweet.Tweet.PublicMetrics.RetweetCount),
		Inline: true,
	})
	fileds = append(fileds, &discordgo.MessageEmbedField{
		Name:   "FavoCount",
		Value:  fmt.Sprintf("%d", btweet.Tweet.PublicMetrics.LikeCount),
		Inline: true,
	})
	fileds = append(fileds, &discordgo.MessageEmbedField{
		Name:   "ReplyCount",
		Value:  fmt.Sprintf("%d", btweet.Tweet.PublicMetrics.ReplyCount),
		Inline: true,
	})

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: content,
		Color:       0x00F1AA,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    btweet.Tweet.Author.Name,
			URL:     userLink,
			IconURL: btweet.Tweet.Author.PofileImageUrl,
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: btweet.Tweet.Author.PofileImageUrl,
		},
		Image: &discordgo.MessageEmbedImage{
			URL: "",
		},
		Fields: fileds,
	}

	if len(btweet.Medias) > 0 {
		embed.Image.URL = btweet.Medias[0].URL
	}

	return embed
}

func makeFilterNotionMessage(arg *CommandArg, manager *BotManager, btweet *BatchTweet) *discordgo.MessageEmbed {
	link := fmt.Sprintf("https://twitter.com/%s/status/%s", btweet.Tweet.Author.Username, btweet.Tweet.Id)
	userLink := fmt.Sprintf("https://twitter.com/%s", btweet.Tweet.Author.Username)
	title := fmt.Sprintf("@%s\tPosted！", btweet.Tweet.Author.Username)

	content := ""
	content += fmt.Sprintf("[Tweet](%s)\n", link)
	content += "────────────────────\n"

	content += "### " + btweet.Tweet.Text

	fileds := make([]*discordgo.MessageEmbedField, 0)
	fileds = append(fileds, &discordgo.MessageEmbedField{
		Name:   "ReTweetCount",
		Value:  fmt.Sprintf("%d", btweet.Tweet.PublicMetrics.RetweetCount),
		Inline: true,
	})
	fileds = append(fileds, &discordgo.MessageEmbedField{
		Name:   "FavoCount",
		Value:  fmt.Sprintf("%d", btweet.Tweet.PublicMetrics.LikeCount),
		Inline: true,
	})
	fileds = append(fileds, &discordgo.MessageEmbedField{
		Name:   "ReplyCount",
		Value:  fmt.Sprintf("%d", btweet.Tweet.PublicMetrics.ReplyCount),
		Inline: true,
	})

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: content,
		Color:       0x00F1AA,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    btweet.Tweet.Author.Name,
			URL:     userLink,
			IconURL: btweet.Tweet.Author.PofileImageUrl,
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: btweet.Tweet.Author.PofileImageUrl,
		},
		Image: &discordgo.MessageEmbedImage{
			URL: "",
		},
		Fields: fileds,
	}

	if len(btweet.Medias) > 0 {
		embed.Image.URL = btweet.Medias[0].URL
	}

	return embed
}

func ConvertMediaMap(medias []xapi.XMedia) map[string]xapi.XMedia {
	mediaMap := make(map[string]xapi.XMedia)
	for _, media := range medias {
		mediaMap[media.MediaKey] = media
	}

	return mediaMap
}

func ConvertBtweet(result *xapi.XTweetsResult, author xapi.XUser, now time.Time) []BatchTweet {
	mediaMap := ConvertMediaMap(result.Inclueds.Medias)
	btweets := make([]BatchTweet, 0)

	for idx, tweet := range result.Data {
		result.Data[idx].Author = author

		medias := make([]xapi.XMedia, 0)
		for _, mediaKey := range tweet.Attochments.MediaKeys {
			media, exist := mediaMap[mediaKey]
			if exist {
				medias = append(medias, media)
			}
		}

		btweet := BatchTweet{
			Tweet:   &result.Data[idx],
			Medias:  medias,
			GetTime: now,
		}
		btweet.Tweet.Author = author
		btweets = append(btweets, btweet)
	}

	return btweets
}

func IsMatchFilter(btweet *BatchTweet, filter *Fileter) bool {

	if filter.MinLikeCount != NULL_FILTER_FILED_INT {
		if btweet.Tweet.PublicMetrics.LikeCount < filter.MinLikeCount {
			return false
		}
	}

	if filter.MaxLikeCount != NULL_FILTER_FILED_INT {
		if btweet.Tweet.PublicMetrics.LikeCount > filter.MaxLikeCount {
			return false
		}
	}

	if filter.MinRetweetCount != NULL_FILTER_FILED_INT {
		if btweet.Tweet.PublicMetrics.RetweetCount < filter.MinRetweetCount {
			return false
		}
	}

	if filter.MaxRetweetCount != NULL_FILTER_FILED_INT {
		if btweet.Tweet.PublicMetrics.RetweetCount > filter.MaxRetweetCount {
			return false
		}
	}

	if len(filter.IncluedUserIDs) > 0 {
		if _, exist := filter.IncluedUserIDs[btweet.Tweet.Author.Id]; !exist {
			return false
		}
	}

	if _, exist := filter.ExcluedUserIDs[btweet.Tweet.Author.Id]; exist {
		return false
	}

	for _, keywords := range filter.AndKeywords {
		if !strings.Contains(btweet.Tweet.Text, keywords) {
			return false
		}
	}

	if len(filter.OrKeywords) > 0 {
		orResult := false
		for _, keywords := range filter.OrKeywords {
			if strings.Contains(btweet.Tweet.Text, keywords) {
				orResult = true
				break
			}
		}
		if !orResult {
			return false
		}
	}

	return true
}
