// パッケージ imgmap はユーザ画像の管理を行うパッケージ。
// 新しいイメージマップを生成し、画像の保存、画像の取得を行うサンプル。
//     m := imgmap.New()
//     m.Set(1, userimagebytes)
//     imgbytes, err := m.Get(1)
package imgmap

import (
	"sync"
	"time"
)

// New は新しいイメージマップを生成する関数。
func New() *ImgMap {
	return &ImgMap{m: make(map[int32]*img)}
}

// ImgMap はイメージマップの構造体。
type ImgMap struct {
	sync.RWMutex
	m map[int32]*img
}

// img はユーザ画像情報の構造体。
// timestamp は画像の有効期間を判定するときに使う。
type img struct {
	data      []byte
	timestamp int64
}

// Get はレシーバから画像データを取得する関数。
// キーに対する画像が登録されていないとき、または有効期間が過ぎているときは、
// 画像が無いことを表す画像を返す。
func (images *ImgMap) Get(key int32) ([]byte, error) {
	now := time.Now().Unix()
	images.RLock()
	i := images.m[key]
	images.RUnlock()

	if i != nil && i.timestamp > now-30 {
		return i.data, nil
	}
	return Asset("noimage.png")
}

// Get はレシーバに画像データを保存する関数。
func (images *ImgMap) Set(key int32, imgdata []byte) {
	i := &img{
		data:      imgdata,
		timestamp: time.Now().Unix(),
	}

	images.Lock()
	images.m[key] = i
	images.Unlock()
}
