package balanced

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHp(t *testing.T) {
	bp := New(3)
	err := bp.Run(NewTask(func() {}, func() time.Duration { return time.Microsecond }))
	assert.Nil(t, err)
}

func TestTooSlow(t *testing.T) {
	bp := New(3)
	_, err1 := bp.RunAsync(NewTask(func() { time.Sleep(time.Hour) }, func() time.Duration { return time.Hour }))
	assert.Nil(t, err1)
	_, err2 := bp.RunAsync(NewTask(func() { time.Sleep(5 * time.Minute) }, func() time.Duration { return 5 * time.Minute }))
	assert.Nil(t, err2)
	_, err3 := bp.RunAsync(NewTask(func() { time.Sleep(30 * time.Second) }, func() time.Duration { return 30 * time.Second }))
	assert.Nil(t, err3)
	_, err4 := bp.RunAsync(NewTask(func() {}, func() time.Duration { return 30 * time.Second }), WithMaxDelay(time.Second))
	assert.Equal(t, ErrWillTakeLonger, err4)
}

func TestCtxCancel(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	bp := New(1, WithContext(ctx))
	_, err1 := bp.RunAsync(NewTask(func() {
		time.Sleep(10 * time.Millisecond)
	}, func() time.Duration { return time.Hour }))
	assert.Nil(t, err1)
	f2, err2 := bp.RunAsync(NewTask(func() {
		time.Sleep(time.Hour)
	}, func() time.Duration { return time.Hour }))
	assert.Nil(t, err2)
	cancelFunc()
	assert.Equal(t, ErrUnprocessed, <-f2)
	_, _ = bp.RunAsync(NewTask(func() {
		time.Sleep(time.Hour)
	}, func() time.Duration { return time.Hour }))
}

func ExamplePool() {
	{
		//new balanced pool with 4 workers
		bp := New(4)
		var err error
		var result int
		//run synchronously until task function is finished
		if err = bp.Run(NewTask(func() {
			time.Sleep(10 * time.Millisecond)
			result = 100
		}, func() time.Duration {
			return 10 * time.Millisecond
		})); err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(result)
	}
	{
		//new balanced pool with 2 workers, and make them busy each with a task that takes 10 ms to run
		//running a new task with maxDelay 1ms will fail immediately, as there is no worker that could make it
		bp := New(2)
		var err error
		//run synchronously until task function is finished
		if _, err = bp.RunAsync(NewTask(func() {
			time.Sleep(10 * time.Millisecond)
		}, func() time.Duration {
			return 10 * time.Millisecond
		})); err != nil {
			fmt.Println(err.Error())
		}
		if _, err = bp.RunAsync(NewTask(func() {
			time.Sleep(10 * time.Millisecond)
		}, func() time.Duration {
			return 10 * time.Millisecond
		})); err != nil {
			fmt.Println(err.Error())
		}
		_, err = bp.RunAsync(NewTask(func() {
		}, func() time.Duration {
			return 10 * time.Microsecond
		}), WithMaxDelay(1*time.Millisecond))
		if err == ErrWillTakeLonger {
			fmt.Println("task wouldn't finish on time")
		} else {
			fmt.Println("err is nil")
		}
	}
	//Output: 100
	//task wouldn't finish on time
}
