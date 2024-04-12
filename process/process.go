package process

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var _uniq int64
var _uniqOnce sync.Once

func LocalUniqID() string {
	_uniqOnce.Do(func() {
		_uniq = time.Now().UnixNano()
	})
	atomic.AddInt64(&_uniq, 1)
	return fmt.Sprintf("%d-%d", os.Getpid(), _uniq)
}
