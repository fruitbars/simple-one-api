package mycommon

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"simple-one-api/pkg/mylog"
)

// 假设 somewhere in your code you have initialized the logger correctly.
// mylog.InitLog("dev") or mylog.InitLog("prod") should have been called.

func CheckStatusCode(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		// Efficiently reading the body and logging in case of an error.
		errMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			mylog.Logger.Error("Failed to read response body",
				zap.Int("status", resp.StatusCode),
				zap.Error(err))
			return errors.New("failed to read error response body")
		}

		// Logging the error with more context.
		mylog.Logger.Error("Unexpected status code",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(errMsg)))

		// Returning a new error with the status code and the body message
		return errors.New(fmt.Sprintf("status %d: %s", resp.StatusCode, string(errMsg)))
	}
	return nil
}
