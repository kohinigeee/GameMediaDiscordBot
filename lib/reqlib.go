package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HeaderParamMap map[string]string

type HeaderParams struct {
	pairs HeaderParamMap
}

func NewHeaderParams() *HeaderParams {
	return &HeaderParams{
		pairs: make(HeaderParamMap),
	}
}

func (params *HeaderParams) AddParam(key string, value string) {
	params.pairs[key] = value
}

func (params *HeaderParams) SetToRequest(req *http.Request) {
	for key, value := range params.pairs {
		req.Header.Set(key, value)
	}
}

func SendGetRequet(url string, params *HeaderParams) ([]byte, error) {

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if params != nil {
		params.SetToRequest(req)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return body, nil
}

func ParseJson(data []byte, target interface{}) error {
	err := json.Unmarshal(data, &target)
	if err != nil {
		return err
	}
	return nil
}
