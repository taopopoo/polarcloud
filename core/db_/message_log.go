package db

import (
	"yunpan/core/config"
	"os"
	"path/filepath"
	"time"
)

/*
	保存消息日志
*/
func SaveMsgLog(name, sendId, content string) {
	tracefile(name, sendId, content)
}

/*
	打印内容到文件中
*/
func tracefile(name, sendId, content string) error {
	fd, err := os.OpenFile(filepath.Join(config.Path_configDir, name), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	fd_time := time.Now().Format("2006-01-02 15:04:05")
	//	fd_content := strings.Join([]string{"======", fd_time, "=====", str_content, "\n"}, "")
	buf := []byte(sendId + " " + fd_time + " " + content + "\r\n")
	fd.Write(buf)
	fd.Close()
	return nil
}
