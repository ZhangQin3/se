package selenium

import (
	"encoding/base64"
	"fmt"
	"gtf/drivers/log"
	"os"
	"time"
)

var debugFlag = true

func setDebug(debug bool) {
	debugFlag = debug
}

func debugLog(format string, args ...interface{}) {
	if !debugFlag {
		return
	}
	log.Textf(format+"\n", args...)
	time.Sleep(10 * time.Microsecond)
}

type QueryError struct {
	Status  int
	Message string
}

func (e QueryError) Error() string {
	return fmt.Sprintf(`{"status":%d, "message":"%s"}`, e.Status, e.Message)
}

func logScreenShot(data *string) (string, error) {
	image := fmt.Sprintf("%d.png", time.Now().Unix())
	dstFile, err := os.Create(image)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}
	defer dstFile.Close()

	d, err := base64.StdEncoding.DecodeString(*data)
	if err != nil {
		return "", err
	}
	dstFile.Write(d)
	log.ToggleImage("Screenshot Image", image, "off")
	return image, nil
}
