package xapi

var (
	XAPI_BASE_URL = "https://api.twitter.com/2"
)

type XTweet struct {
	EditHistoryTweetIds []string `json:"edit_history_tweet_ids"`
	Id                  string   `json:"id"`
	Text                string   `json:"text"`
	Author              XUser
}

type XUser struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type XTweetResult struct {
	Data []XTweet `json:"data"`
}

type XUserResult struct {
	Data XUser `json:"data"`
}
