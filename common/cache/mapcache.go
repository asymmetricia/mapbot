package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
)

var cacheMu = sync.Mutex{}

type CacheEntry struct {
	Version int
	Image   image.Image
}

func (c *CacheEntry) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["version"] = strconv.Itoa(c.Version)
	buf := &bytes.Buffer{}
	png.Encode(buf, c.Image)
	m["png"] = base64.StdEncoding.EncodeToString(buf.Bytes())
	return json.Marshal(m)
}

func (c *CacheEntry) UnmarshalJSON(data []byte) error {
	m := make(map[string]interface{})
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	v, ok := m["version"]
	if !ok {
		return errors.New("missing `version`")
	}
	vStr, ok := v.(string)
	if !ok {
		return fmt.Errorf("`version` was %T, not string", v)
	}
	vInt, err := strconv.Atoi(vStr)
	if err != nil {
		return fmt.Errorf("`version` was %q, not parseable int: %s", vStr, err)
	}
	c.Version = vInt

	pngData, ok := m["png"]
	if !ok {
		return errors.New("missing `png`")
	}
	pngb64, ok := pngData.(string)
	if !ok {
		return fmt.Errorf("`png` was %T, not string", pngData)
	}

	pngBytes, err := base64.StdEncoding.DecodeString(pngb64)
	if err != nil {
		return fmt.Errorf("decoding `png` from base64: %s", err)
	}

	pngBuf := bytes.NewBuffer(pngBytes)
	img, err := png.Decode(pngBuf)
	if err != nil {
		return fmt.Errorf("`png` couldn't decode as png: %s", err)
	}
	c.Image = img
	return nil
}

var log = mbLog.Log

var CacheDir = flag.String("cache-dir", "/tmp/mapbot", "directory to store cached map files")

func Get(key string) (ret *CacheEntry, ok bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	keyHash := sha256.Sum256([]byte(key))
	key = hex.EncodeToString(keyHash[:])

	f, err := os.Open(*CacheDir + "/" + key)

	var data []byte

	if err == nil {
		defer f.Close()

		data, err = ioutil.ReadAll(f)
	}

	if err == nil {
		ret = new(CacheEntry)
		err = json.Unmarshal(data, ret)
	}

	if err == nil {
		return ret, true
	}

	// err wasn't nil
	if !os.IsNotExist(err) {
		log.Errorf("error opening cached file %s: %s", key, err)
	} else {
		log.Debugf("cache miss %s", key)
	}
	return nil, false
}

func Put(key string, entry *CacheEntry) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	keyHash := sha256.Sum256([]byte(key))
	key = hex.EncodeToString(keyHash[:])

	_, err := os.Stat(*CacheDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(*CacheDir, os.FileMode(0755))
	}

	var f *os.File
	if err == nil {
		f, err = os.OpenFile(*CacheDir+"/"+key, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.FileMode(0644))
	}
	if err == nil {
		defer f.Close()
		enc := json.NewEncoder(f)
		err = enc.Encode(entry)
	}
	if err != nil {
		log.Errorf("writing cached file %s: %s", key, err)
	}
}
