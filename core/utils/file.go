package utils

import (
	"encoding/json"
	"os"
)

/*
	检查目录是否存在，不存在则创建
*/
func CheckCreateDir(dir_path string) {
	if ok, err := PathExists(dir_path); err == nil && !ok {
		Mkdir(dir_path)
	}
}

/*
	判断一个路径的文件是否存在
*/
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

/*
	递归创建目录
*/
func Mkdir(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
	//	err := os.Mkdir(path, os.ModeDir)
	if err != nil {
		//		fmt.Println("创建文件夹失败", path, err)
		return err
	}
	return nil
}

/*
	保存对象为json格式
*/
func SaveJsonFile(name string, o interface{}) error {
	bs, err := json.Marshal(o)
	if err != nil {
		return err
	}
	return SaveFile(name, &bs)
}

/*
	保存文件
*/
func SaveFile(name string, bs *[]byte) error {
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, os.ModePerm)
	//	file, err := os.Create(filepath.Join(gconfig.Store_dir, hashName))
	if err != nil {
		file.Close()
		return err
	}
	_, err = file.Write(*bs)
	if err != nil {
		file.Close()
		return err
	}
	file.Close()
	return nil
}
