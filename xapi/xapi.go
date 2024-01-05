package xapi

import (
	"fmt"
	"gamemediabot/lib"
	"strings"
	"time"
)

type XClinet struct {
	XBearer string
}

func NewXClinet(xBearer string) *XClinet {
	return &XClinet{
		XBearer: xBearer,
	}
}

func NewDefaultHeaderParams() *lib.HeaderParams {
	params := lib.NewHeaderParams()
	params.AddParam("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	return params
}

func (client *XClinet) GetUser(userName string) (XUser, error) {
	url := fmt.Sprintf("%s/users/by/username/%s?user.fields=profile_image_url", XAPI_BASE_URL, userName)
	fmt.Println("url: ", url)
	params := NewDefaultHeaderParams()
	params.AddParam("Authorization", fmt.Sprintf("Bearer %s", client.XBearer))

	dummyUser := XUser{}

	body, err := lib.SendGetRequet(url, params)
	if err != nil {
		return dummyUser, err
	}

	var result XUserResult
	err = lib.ParseJson(body, &result)
	if err != nil {
		return dummyUser, err
	}

	if result.Data.Id == "" {
		err = fmt.Errorf("user not found: %s", userName)
		return dummyUser, err
	}

	fmt.Printf("GetUser: %+v", result.Data)

	return result.Data, nil
}

func (client *XClinet) GetUserByID(userID string) (XUser, error) {
	url := fmt.Sprintf("%s/users/%s?user.fields=profile_image_url", XAPI_BASE_URL, userID)
	params := NewDefaultHeaderParams()
	params.AddParam("Authorization", fmt.Sprintf("Bearer %s", client.XBearer))

	dummyUser := XUser{}

	body, err := lib.SendGetRequet(url, params)
	if err != nil {
		return dummyUser, err
	}

	var result XUserResult
	err = lib.ParseJson(body, &result)
	if err != nil {
		return dummyUser, err
	}

	if result.Data.Id == "" {
		err = fmt.Errorf("user not found: %s", userID)
		return dummyUser, err
	}

	return result.Data, nil
}

func (client *XClinet) GetTweetsByUserId(user *XUser, maxResults int, startDate *time.Time) (XTweetsResult, error) {
	url := fmt.Sprintf("%s/users/%s/tweets?max_results=%d&tweet.fields=created_at,author_id,public_metrics,attachments&media.fields=url&expansions=attachments.media_keys", XAPI_BASE_URL, user.Id, maxResults)
	if startDate != nil {
		dateStr := startDate.Format(time.RFC3339)
		url += fmt.Sprintf("&start_time=%s", dateStr)
	}

	params := NewDefaultHeaderParams()
	params.AddParam("Authorization", fmt.Sprintf("Bearer %s", client.XBearer))

	body, err := lib.SendGetRequet(url, params)

	var result XTweetsResult
	err = lib.ParseJson(body, &result)
	if err != nil {
		return result, err
	}

	for idx := range result.Data {
		result.Data[idx].Author = *user
	}

	// fmt.Printf("GetTweetsByUserId: %+v\n\n", result)

	return result, nil
}

func (client *XClinet) GetTweetsByID(tweetIDs []string) (XTweetsResult, error) {
	ids := strings.Join(tweetIDs, ",")
	url := fmt.Sprintf("%s/tweets?ids=%s&tweet.fields=created_at,author_id,public_metrics,attachments&media.fields=url&expansions=attachments.media_keys", XAPI_BASE_URL, ids)

	params := NewDefaultHeaderParams()
	params.AddParam("Authorization", fmt.Sprintf("Bearer %s", client.XBearer))

	body, err := lib.SendGetRequet(url, params)

	var result XTweetsResult
	err = lib.ParseJson(body, &result)
	if err != nil {
		return result, err
	}

	// fmt.Printf("GetTweetsByID: %+v\n\n", result)

	return result, nil
}

func (client *XClinet) GetTweetInfo(tweetId string, startDate *time.Time) (XTweetResult, error) {
	url := fmt.Sprintf("%s/tweets/%s?tweet.fields=created_at,author_id,public_metrics,attachments&media.fields=url&expansions=attachments.media_keys", XAPI_BASE_URL, tweetId)

	params := NewDefaultHeaderParams()
	params.AddParam("Authorization", fmt.Sprintf("Bearer %s", client.XBearer))

	body, err := lib.SendGetRequet(url, params)

	var result XTweetResult
	err = lib.ParseJson(body, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
