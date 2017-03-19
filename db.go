package main

import (
	"fmt"
	"time"

	"encoding/json"

	"github.com/boltdb/bolt"
)

type db struct {
	db    *bolt.DB
	inbox chan func()
}

var (
	lgtmBucket = []byte("lgtm")
)

func OpenDB(path string) (*db, error) {
	bdb, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		return nil, err
	}

	db := &db{
		db:    bdb,
		inbox: make(chan func(), 1),
	}
	go db.Serve()

	err = db.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(lgtmBucket)
		return err
	})
	if err != nil {
		db.db.Close()
		return nil, err
	}

	return db, nil
}

func (db *db) Serve() {
	select {}
}

func (db *db) Close() error {
	return db.db.Close()
}

func (db *db) LGTM(pr int, user string) {
	key := []byte(fmt.Sprintf("pr-%d", pr))
	//db.inbox <- func() {
	db.db.Update(func(tx *bolt.Tx) error {
		var curLGTM []string
		curVal := tx.Bucket(lgtmBucket).Get(key)
		if curVal != nil {
			json.Unmarshal(curVal, &curLGTM) // ignore error
		}
		for _, ex := range curLGTM {
			if ex == user {
				// we're done
				return nil
			}
		}
		curLGTM = append(curLGTM, user)
		bs, _ := json.Marshal(curLGTM)
		return tx.Bucket(lgtmBucket).Put(key, bs)
	})
	//}
}

func (db *db) LGTMs(pr int) []string {
	key := []byte(fmt.Sprintf("pr-%d", pr))
	var lgtms []string
	db.db.View(func(tx *bolt.Tx) error {
		bs := tx.Bucket(lgtmBucket).Get(key)
		if bs == nil {
			return nil
		}
		json.Unmarshal(bs, &lgtms)
		return nil
	})
	return lgtms
}
