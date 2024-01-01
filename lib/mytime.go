package lib

import (
	"fmt"
	"time"
)

func UTCtimeToLoaclTime(date time.Time) (string, error) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	dummy := ""

	if err != nil {
		fmt.Println("LoadLocation Error:", err)
		return dummy, err
	}

	localTime := date.In(loc)

	return localTime.Format("2006-01-02 15:04:05"), nil
}
