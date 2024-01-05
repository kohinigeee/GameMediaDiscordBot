package xapi

var (
	XAPI_BASE_URL = "https://api.twitter.com/2"
)

type XTweet struct {
	EditHistoryTweetIds []string            `json:"edit_history_tweet_ids"`
	Id                  string              `json:"id"`
	Text                string              `json:"text"`
	AuthorID            string              `json:"author_id"`
	PublicMetrics       XTweetPublicMetrics `json:"public_metrics"`
	Attochments         XTweetAttachments   `json:"attachments"`
	Author              XUser
}

type XUser struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Username       string `json:"username"`
	PofileImageUrl string `json:"profile_image_url"`
}

type XTweetsResult struct {
	Data     []XTweet      `json:"data"`
	Inclueds XTweetInclued `json:"includes"`
}

type XTweetResult struct {
	Data     XTweet        `json:"data"`
	Inclueds XTweetInclued `json:"includes"`
}

type XUserResult struct {
	Data XUser `json:"data"`
}

type XTweetPublicMetrics struct {
	RetweetCount    int `json:"retweet_count"`
	ReplyCount      int `json:"reply_count"`
	LikeCount       int `json:"like_count"`
	QuoteCount      int `json:"quote_count"`
	BookmarkCount   int `json:"bookmark_count"`
	ImpressionCount int `json:"impression_count"`
}

type XTweetInclued struct {
	Medias []XMedia `json:"media"`
}

type XMedia struct {
	MediaKey string `json:"media_key"`
	Type     string `json:"type"`
	URL      string `json:"url"`
}

type XTweetAttachments struct {
	MediaKeys []string `json:"media_keys"`
}
