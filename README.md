# monosafe
Package provides a couple of single-value in-memory caches with auto- and manual validation controls.

Anticipated workloads are read-heavy, with none-to-little writes.  
e.g. caching smaller lookup tables or API responses.

Prefer `LockFree` implementation over `Transact` if you don't
need consistent views or partial updates.

Usage: initialise the implementation of choice and run.

```go
package main

import (
	"context"
	"iter"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kelindar/bitmap"
	"github.com/kzmnbrs/monosafe"
)

type ATM struct {
	ID       uint32
	Title    string
	Location struct {
		Lat float64
		Lon float64
	}
}

type ATMs struct {
	idToATM      map[uint32]*ATM
	bmOpen24     bitmap.Bitmap
	bmCanEatCash bitmap.Bitmap
}

var queryPool = sync.Pool{
	New: func() any {
		return bitmap.Bitmap{}
	},
}

func (t *ATMs) Filter(open24, canEatCash bool) iter.Seq[*ATM] {
	return func(yield func(*ATM) bool) {
		query := queryPool.Get().(bitmap.Bitmap)
		query.Clear()

		query.Grow(uint32(len(t.idToATM)))
		query.Ones()

		if open24 {
			query.And(t.bmOpen24)
		}
		if canEatCash {
			query.And(t.bmCanEatCash)
		}

		yieldBreak := false
		query.Range(func(id uint32) {
			if yieldBreak {
				return
			}
			yieldBreak = yield(t.idToATM[id])
		})
	}
}

func main() {
	sigint, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	changeDataCapture := make(chan struct{})

	atmSafe := monosafe.MustLockFree[ATMs](
		func(ctx context.Context, oldValue *ATMs) (*ATMs, error) {
			// Query DB and all. You can return the old value
		},
	)
	if err := atmSafe.Run(sigint,
		monosafe.WithManualControl(changeDataCapture),
		monosafe.WithTick(time.Minute*5),
	); err != nil {
		log.Fatal("run atm safe: ", err)
	}

	http.DefaultServeMux.HandleFunc("/atm", func(w http.ResponseWriter, r *http.Request) {
		iter := atmSafe.Load().Filter(r.Form.Has("open24h"), r.Form.Has("can_eat_cash"))
		// ...
	})
}
```
