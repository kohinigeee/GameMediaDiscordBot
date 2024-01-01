package botmanager

import (
	"fmt"
	"gamemediabot/lib"
	"gamemediabot/xapi"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	globalManger *BotManager = nil
)

type BotManager struct {
	allowedChannels   map[string]bool
	targetUsers       map[string]xapi.XUser
	client            *xapi.XClinet
	discordgoSession  *discordgo.Session
	batchDurationMinu int
	lastBatchDate     time.Time
	nextBatchDate     time.Time
	isRunningBatch    bool
}

func SetGlobalManager(manager *BotManager) {
	globalManger = manager
}

func GetGlobalManager() *BotManager {
	return globalManger
}

func NewBotManager(discordgoSession *discordgo.Session, client *xapi.XClinet, batchDurationMinu int) *BotManager {

	manager := &BotManager{
		allowedChannels:   make(map[string]bool),
		targetUsers:       make(map[string]xapi.XUser),
		client:            client,
		discordgoSession:  discordgoSession,
		batchDurationMinu: batchDurationMinu,
		lastBatchDate:     time.Now().UTC(),
		isRunningBatch:    false,
	}

	manager.nextBatchDate = manager.lastBatchDate.Add(time.Duration(batchDurationMinu) * time.Minute)

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

	manager.discordgoSession.AddHandler(onDiscordMessageCreate)
	return manager
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

func (manager *BotManager) getTweets() []xapi.XTweet {
	maxResults := 5
	tweets := make([]xapi.XTweet, 0)
	//同時刻のツイートを取得しないように、最後に取得した時間の2秒後から取得する
	startTime := manager.lastBatchDate.Add(2 * time.Second)

	for _, user := range manager.targetUsers {
		gotTweets, err := manager.client.GetTweetsByUserId(&user, maxResults, &startTime)
		if err != nil {
			continue
		}
		tweets = append(tweets, gotTweets...)
	}

	return tweets
}

func (manager *BotManager) sentBatchMessages(channelId string, tweets []xapi.XTweet) {
	for idx, tweet := range tweets {
		author := tweet.Author
		link := fmt.Sprintf("https://twitter.com/%s/status/%s", author.Username, tweet.Id)
		msg := ""
		if idx > 0 {
			msg += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
		}
		msg += fmt.Sprintf("### 【 @%s Posted！ ([Tweet](%s)) 】\n", author.Username, link)
		msg += ">>> "
		msg += tweet.Text
		manager.discordgoSession.ChannelMessageSend(channelId, msg)
		log.Printf("channelId = %s\ntweet = %s\n", channelId, msg)
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
		}
	}
}

func (manager *BotManager) Start() {
	if manager.isRunningBatch {
		return
	}
	manager.isRunningBatch = true
	go manager.batchLoop()
}

func (manager *BotManager) OnMessageCreate(s *discordgo.Session, channelID, msg string) {
	_, err := s.ChannelMessageSend(channelID, msg)
	if err != nil {
		log.Println("Error sending message")
	}
}

func onDiscordMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	commandTriger := "!mediabot"
	msg := m.Content
	tokens := strings.Split(msg, " ")
	if commandTriger != tokens[0] {
		return
	}

	if len(tokens) < 2 {
		onInvalidCommand(s, m)
	}

	commandName := tokens[1]
	manager := GetGlobalManager()

	//コマンドの実行
	switch commandName {
	case "allow":
		onAllowChannelCommand(s, m, manager)
	case "addtarget":
		onAddTargetUserCommand(s, m, manager, tokens)
	case "help":
		onHelpCommand(s, m)
	case "change_duration":
		onChangeBatchDurationCommand(s, m, manager, tokens)
	case "targets":
		onTargetsCommand(s, m, manager)
	case "settings":
		onSettingsCommand(s, m, manager)
	case "test":
		onTestCommand(s, m, manager)
	default:
		onInvalidCommand(s, m)
	}
}

func sendMessage(s *discordgo.Session, channelID string, msg string) {
	_, err := s.ChannelMessageSend(channelID, msg)
	if err != nil {
		errmsg := fmt.Sprintf("Error sending message : cnannelId[%s], msg[%s]", channelID, msg)
		log.Println(errmsg)
	}
}

func onAllowChannelCommand(s *discordgo.Session, m *discordgo.MessageCreate, manager *BotManager) {
	channelId := m.ChannelID
	manager.AddAllowedChannel(channelId)

	sendMessage(s, channelId, "このチャンネルでの発言を許可しました")
}

func onChangeBatchDurationCommand(s *discordgo.Session, m *discordgo.MessageCreate, manager *BotManager, tokens []string) {
	maxDuration := 120
	minDuariont := 1

	if len(tokens) < 3 {
		errmsg := "分を指定してください"
		sendMessage(s, m.ChannelID, errmsg)
		return
	}

	minu, err := strconv.Atoi(tokens[2])
	if err != nil {
		errmsg := "分には数値を指定してください(Max : 120, Min : 1)"
		sendMessage(s, m.ChannelID, errmsg)
		return
	}

	if minu >= maxDuration || minu < minDuariont {
		errmsg := fmt.Sprintf("分の値は%d以上%d以下にしてください", minDuariont, maxDuration)
		sendMessage(s, m.ChannelID, errmsg)
		return
	}

	oldDuarion := manager.batchDurationMinu
	manager.setBatchDurationMinu(minu)
	msg := fmt.Sprintf("バッチ処理の間隔を%d(分)から%d(分)に変更しました\n次回のバッチ処理後に適用されます", oldDuarion, minu)
	sendMessage(s, m.ChannelID, msg)
}

func onAddTargetUserCommand(s *discordgo.Session, m *discordgo.MessageCreate, manager *BotManager, tokens []string) {
	if len(tokens) < 3 {
		errmsg := "ユーザー名を指定してください"
		sendMessage(s, m.ChannelID, errmsg)
		return
	}

	userName := tokens[2]
	user, err := manager.client.GetUser(userName)

	if err != nil {
		errmsg := fmt.Sprintf("ユーザー名[%s]のユーザーが見つかりませんでした", userName)
		sendMessage(s, m.ChannelID, errmsg)
		return
	}

	manager.AddTargetUser(user)
	msg := fmt.Sprintf("ユーザー名[%s]のユーザーを追加しました", user.Username)
	sendMessage(s, m.ChannelID, msg)

	for id, target := range manager.targetUsers {
		fmt.Printf("key = %s : user = %+v\n", id, target)
	}
}

func onTargetsCommand(s *discordgo.Session, m *discordgo.MessageCreate, manager *BotManager) {
	msg := "登録済みのユーザー一覧\n"
	for _, user := range manager.targetUsers {
		link := fmt.Sprintf("https://twitter.com/%s", user.Username)
		msg += fmt.Sprintf("%s  (%s)\n", user.Username, link)
	}
	sendMessage(s, m.ChannelID, msg)
}

func onSettingsCommand(s *discordgo.Session, m *discordgo.MessageCreate, manager *BotManager) {
	msg := "設定値一覧\n"
	msg += fmt.Sprintf("バッチ処理の間隔 : %d(分)\n", manager.batchDurationMinu)

	nextLoaclTime := lib.UTCtimeToLoaclTime(manager.GetNextBatchTime())
	msg += fmt.Sprintf("次回のバッチ処理 : %s\n", nextLoaclTime)

	sendMessage(s, m.ChannelID, msg)
}

func onHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	commands := []string{
		"help",
		"allow",
		"change_duration <分>",
		"settings",
		"targes",
		"settings",
		"addtarget <ツイッターユーザID(@は除外)>",
	}

	msg := "コマンド一覧( 先頭に!mediabotを付けてください)\n"
	for _, command := range commands {
		msg += command + "\n"
	}
	sendMessage(s, m.ChannelID, msg)
}

func onInvalidCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	msg := "不正なコマンドです。コマンド一覧は「!mediabot help」で確認できます。"
	sendMessage(s, m.ChannelID, msg)
}

func onTestCommand(s *discordgo.Session, m *discordgo.MessageCreate, manager *BotManager) {
}
