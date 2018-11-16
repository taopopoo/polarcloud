package db

import (
	"sync"
)

var friendIdsLock = new(sync.RWMutex)
var friendIds = make(map[string]string)

func loadFriends() error {
	fs, err := Friends_getall()
	if err != nil {
		return err
	}
	for _, one := range fs {
		friendIds[one.Id] = one.Id
	}
	return nil
}

type Friends struct {
	Id   string
	Name string
}

func Friends_add(id string) error {
	stmt, err := db.Prepare("insert into friends values(?,?)")
	if err != nil {
		return err
	}
	stmt.Exec(id, "")
	friendIdsLock.Lock()
	friendIds[id] = id
	friendIdsLock.Unlock()
	return nil
}

func Friends_getall() ([]Friends, error) {
	friends := make([]Friends, 0)
	rows, err := db.Query("select * from friends")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var id string
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			break
		}
		friend := Friends{
			Id:   id,
			Name: name,
		}
		friends = append(friends, friend)
	}
	return friends, err
}

/*
	检查用户id是否存在
*/
func Friends_findIdExist(id string) (ok bool) {
	friendIdsLock.Lock()
	_, ok = friendIds[id]
	friendIdsLock.Unlock()
	return
}
