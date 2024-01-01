package xapi

import (
	"fmt"
	"gamemediabot/lib"
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
	url := fmt.Sprintf("%s/users/by/username/%s", XAPI_BASE_URL, userName)
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

	return result.Data, nil
}

func (client *XClinet) GetTweetsByUserId(userId string, maxResults int, startDate *time.Time) ([]XTweet, error) {
	url := fmt.Sprintf("%s/users/%s/tweets?max_results=%d", XAPI_BASE_URL, userId, maxResults)
	if startDate != nil {
		dateStr := startDate.Format(time.RFC3339)
		url += fmt.Sprintf("&start_time=%s", dateStr)
	}

	params := NewDefaultHeaderParams()
	params.AddParam("Authorization", fmt.Sprintf("Bearer %s", client.XBearer))

	body, err := lib.SendGetRequet(url, params)

	var result XTweetResult
	err = lib.ParseJson(body, &result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}
